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