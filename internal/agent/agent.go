package agent

import (
	"AmiyaAgent/internal/component"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/coze-dev/cozeloop-go"
)

func NewDeepAgent(ctx context.Context, model *openai.ChatModel, agentRoot string,cozeloopApiToken string,cozeloopWorkspaceID string) (adk.Agent, error) {
	// 创建LocalBackend Tools 后端工具实例
	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, err
	}

	// handler := callbacks.NewHandlerHelper().
    // OnStart(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
    //     log.Printf("[trace] %s/%s start", info.Component, info.Name)
    //     return ctx
    // }).
    // OnEnd(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
    //     log.Printf("[trace] %s/%s end", info.Component, info.Name)
    //     return ctx
    // }).
    // OnError(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
    //     log.Printf("[trace] %s/%s error: %v", info.Component, info.Name, err)
    //     return ctx
    // }).
    // Handler()

	// // 注册为全局 Callback
	// callbacks.AppendGlobalHandlers(handler)

	handler := callbacks.NewHandlerBuilder().
	OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		log.Printf("[trace] %s/%s start", info.Component, info.Name) 
		 return ctx 
	}).
		OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context { 
			log.Printf("[trace] %s/%s end", info.Component, info.Name) 
			return ctx 
	}).
		OnErrorFn(func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context { 
			log.Printf("[trace] %s/%s error: %v", info.Component, info.Name, err) 
			return ctx 
	}).Build()

	
	callbacks.AppendGlobalHandlers(handler)
	
	// 配制 CozeLoop 追踪
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			return nil, fmt.Errorf("初始化CozeLoop客户端失败: %v", err)
		}
		defer func() {
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
		callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
		log.Println("CozeLoop追踪已启用")
	} else {
		log.Println("CozeLoop追踪未启用，缺少API Token或Workspace ID")
	}

	// 创建自定义工具集
	tools := InitTools()
	toolsConfig := adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	}

	// 获取文件系统操作说明
	extInstruction := FileSystemInstruction(agentRoot)

	// 创建ChatModelAgent类型的Agent实例
	// agentConfig := &adk.ChatModelAgentConfig{
	// 	Name: agent.AmiyaName,
	// 	Description: agent.AmiyaDescription,
	// 	Instruction: agent.AmiyaInstruction,
	// 	Model: chatModel,
	// }
	// agent,err := adk.NewChatModelAgent(ctx,agentConfig)
	// if err != nil {
	// 	log.Fatal("创建 ChatModelAgent 实例失败:", err)
	// }
	// log.Println("ChatModelAgent 实例创建成功")

	// 创建DeepAgent类型的Agent实例
	agentConfig := &deep.Config{
		Name:           AmiyaName,
		Description:    AmiyaDescription,
		Instruction:    AmiyaInstruction + "\n\n" + extInstruction, // 将文件操作说明添加到系统提示词中
		ChatModel:      model,
		ToolsConfig:    toolsConfig,
		Backend:        backend, // 注入LocalBackend工具集
		StreamingShell: backend, // 支持流式 Shell 输出
		MaxIteration:   50,      // 最大思考/工具调用循环次数
		// 注册安全工具中间件，用于捕获和处理工具调用错误
		Handlers: []adk.ChatModelAgentMiddleware{
			&component.ApprovalMiddleware{}, // 处理工具调用中的用户审批流程
			&component.SafeToolMiddleware{}, // 捕获工具调用错误
		},
		// 配置模型重试策略，处理速率限制错误
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 5,
			IsRetryAble: func (_ context.Context,err error) bool{
				return strings.Contains(err.Error(), "429") ||
					strings.Contains(err.Error(), "Too Many Requests") ||
					strings.Contains(err.Error(), "qpm limit")
			},
		},
	}
	agent, err := deep.New(ctx, agentConfig)
	if err != nil {
		return nil, err
	}

	return agent, nil
}