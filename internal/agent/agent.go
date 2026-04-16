package agent

import (
	"AmiyaAgent/internal/component"
	"context"我的
	"strings"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/compose"
)

func NewDeepAgent(ctx context.Context, model *openai.ChatModel, agentRoot string) (adk.Agent, error) {
	// 创建LocalBackend Tools 后端工具实例
	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, err
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
			&component.SafeToolMiddleware{},
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