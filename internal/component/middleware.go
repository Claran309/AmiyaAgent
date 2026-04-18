package component

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
)

// safeToolMiddleware 是一个自定义中间件，用于安全地处理工具调用错误
// 它将工具执行错误转换为可读的错误消息，而不是中断整个流程
type SafeToolMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

// WrapInvokableToolCall 包装同步工具调用，捕获错误并返回格式化的错误消息
func (m *SafeToolMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	// 中间件函数逻辑
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		// 调用原始工具执行函数，并捕获任何错误
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			// 如果是中断重新运行错误，直接返回以允许 Agent 重试
			if _, ok := compose.IsInterruptRerunError(err);ok {
				return "", err
			}
			// 否则，返回一个格式化的错误消息
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		return result, nil
	}, nil
}

// WrapStreamableToolCall 包装流式工具调用，安全地处理流中的错误
func (m *SafeToolMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	// 中间件函数逻辑
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		// 调用原始流式工具执行函数，并捕获任何错误
		stream, err := endpoint(ctx, args, opts...)
		if err != nil {
			// 如果是中断重新运行错误，直接返回以允许 Agent 重试
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			// 返回包含错误消息的单块读取器
			return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
		}
		// 包装原始读取器以处理流中的错误
		return safeWrapReader(stream), nil

	}, nil
}

// singleChunkReader 创建一个只包含单个消息的流读取器
func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}

// safeWrapReader 包装流读取器，捕获流中的错误并转换为错误消息
func safeWrapReader(sr *schema.StreamReader[string]) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](64)
	go func() {
		defer w.Close()
		for {
			chunk, err := sr.Recv()
			// 正常结束流
			if errors.Is(err, io.EOF) {
				return
			}
			// 流中发生错误，发送错误消息
			if err != nil {
				_ = w.Send(fmt.Sprintf("\n[tool error] %v", err), nil)
				return
			}
			// 转发正常的数据块
			_ = w.Send(chunk, nil)
		}
	}()
	return r
}

// approvalMiddleware 处理工具调用中的用户审批流程
type ApprovalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

// WrapInvokableToolCall 包装同步工具调用，实现中断-恢复流程
func (m *ApprovalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	// 只对 "execute" 工具进行审批拦截，其他工具直接通过
	// if tCtx.Name != "execute" {
	// 	return endpoint, nil
	// }

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		// 第一次执行：检查是否已被中断过
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

		if !wasInterrupted {
			// 首次执行，还未被中断过
			// 触发 StatefulInterrupt： 保存args，返回审批信息，暂停执行
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		// 恢复执行：获取用户的审批结果
		isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)

		if isTarget && hasData {
			if data.Approved {
				// 用户批准，继续执行工具
				return endpoint(ctx, storedArgs, opts...)
			}
			// 用户拒绝，返回拒绝消息
			if data.DisapproveReason != nil {
				return fmt.Sprintf("工具 '%s' 审批被拒绝: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("工具 '%s' 审批被拒绝", tCtx.Name), nil
		}

		// 检查是否还有其他中断点未处理
		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if isTarget {
			// 还有其他中断点未处理
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		// 所有中断点都已处理，执行工具
		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

// WrapStreamableToolCall 包装流式工具调用，实现中断-恢复流程
func (m *ApprovalMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	// 只对 "execute" 工具进行审批拦截，其他工具直接通过
	// if tCtx.Name != "execute" {
	// 	return endpoint, nil
	// }

	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		// 第一次执行：检查是否已被中断过
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			// 首次执行，触发 StatefulInterrupt： 保存args，返回审批信息，暂停执行
			return nil, tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		// 恢复执行：获取用户的审批结果
		isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				// 用户批准，继续执行工具
				return endpoint(ctx, storedArgs, opts...)
			}
			// 用户拒绝，返回包含拒绝消息的单块读取器
			if data.DisapproveReason != nil {
				return singleChunkReader(fmt.Sprintf("工具 '%s' 审批被拒绝: %s", tCtx.Name, *data.DisapproveReason)), nil
			}
			return singleChunkReader(fmt.Sprintf("工具 '%s' 审批被拒绝", tCtx.Name)), nil
		}

		// 检查是否还有其他中断点未处理
		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if isTarget {
			// 还有其他中断点未处理，继续中断当前点
			return nil, tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		// 所有中断点都已处理，执行工具
		return endpoint(ctx, storedArgs, opts...)
	}, nil
}