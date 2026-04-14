package main

import (
	"AmiyaAgent/internal/agent"
	"AmiyaAgent/internal/component"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)



func main() {
	// 初始化上下文
	ctx := context.Background()

	// 获取apiKey等环境变量并封装为config
	err := godotenv.Load()
	if err != nil {
		log.Println("未找到 .env 文件，使用系统环境变量")
	}
	apiKey, baseURL, modelName, sessionDir,agentRoot := os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_MODEL_NAME"), os.Getenv("SESSION_DIR"), os.Getenv("AGENT_ROOT")
	if apiKey == "" || baseURL == "" || modelName == "" || sessionDir == "" || agentRoot == "" {
		log.Fatal("请设置环境变量")
	}

	// 创建 ChatModel 实例
	chatModel, err := component.NewChatModel(ctx, apiKey, baseURL, modelName)
	if err != nil {
		log.Fatal("创建 ChatModel 实例失败:", err)
	}
	log.Println("ChatModel 实例创建成功")

	// 获取Agent绝对路径
	if abs, err := filepath.Abs(agentRoot); err == nil {
		agentRoot = abs
	}

	// 创建 DeepAgent 实例
	agent, err := agent.NewDeepAgent(ctx, chatModel, agentRoot)
	if err != nil {
		log.Fatal("创建 DeepAgent 实例失败:", err)
	}
	log.Println("DeepAgent 实例创建成功")

	// 创建agentRunner实例
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})
	log.Println("AgentRunner 实例创建成功")

	// 创建会话存储
	store, err := component.NewStore(sessionDir)
	if err != nil {
		log.Fatal("创建会话存储失败:", err)
	}
	log.Println("会话存储创建成功")

	var sessionID string
	fmt.Print("请输入会话ID（留空则创建新会话）: ")
	fmt.Scanln(&sessionID)

	// 处理会话 ID
	if sessionID == "" { // 如果为空，生成新 UUID
		sessionID = uuid.New().String()
		fmt.Printf("创建新会话: %s\n", sessionID)
	} else {
		fmt.Printf("恢复会话: %s\n", sessionID)
	}

	// 获取或创建会话
	session, err := store.GetSession(sessionID)
	if err != nil {
		log.Fatal("获取或创建会话失败:", err)
	}
	log.Println("会话获取或创建成功")

	// 启动对话
	fmt.Println()
	fmt.Println("===============================================开始对话（当前会话标题：" + session.GetTitle() + "）=======================================================")
	fmt.Println()
	fmt.Println("阿米娅: 博士，您好！我是阿米娅，有什么需要我帮忙的吗？")
	fmt.Println()

	//log.Println("")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("博士：")
		if !scanner.Scan() {
			break
		}

		fmt.Println()

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// 封装用户消息
		userMsg := schema.UserMessage(input)
		if err := session.Append(userMsg); err != nil { // 将消息保存到会话
			log.Println("保存用户消息失败:", err)
			continue
		}

		// 获取当前会话的消息历史
		messages := session.GetMessages()

		// 调用 AgentRunner 生成回复
		fmt.Print("阿米娅：")
		events := runner.Run(ctx, messages)
		content, err := getAssistantMsg(events)
		if err != nil {
			log.Println("生成回复失败:", err)
			// 回滚消息历史
			messages = messages[:len(messages)-1]
			continue
		}

		// 将模型回复加入对话历史
		err = session.Append(schema.AssistantMessage(content, nil))
		if err != nil {
			log.Println("保存助手消息失败:", err)
			continue
		}

		fmt.Println()
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	
	fmt.Printf("\n\n会话已保存，会话ID：%s\n", sessionID)
}

func getAssistantMsg(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	// 创建一个字符串构建器，用于累积所有回复内容
	var builder strings.Builder

	// 循环遍历事件迭代器中的所有事件
	for {
		// 从迭代器中获取下一个事件
		event, ok := events.Next()
		if !ok {
			break
		}

		// 检查事件中是否包含错误
		if event.Err != nil {
			return "", event.Err
		}

		// 检查事件的输出是否为空或消息输出为空
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		// 获取消息输出对象，简化后续代码
		msg := event.Output.MessageOutput

		// 若本次消息为工具调用结果，则提取工具输出内容并打印（不累积到最终回复中）
		if msg.Role == schema.Tool {
			content := getToolMsg(msg)
			fmt.Printf("[调用工具结果] %s\n", truncate(content, 200))
			continue
		}

		// 只处理Assistant消息
		if msg.Role != schema.Assistant {
			continue
		}

		// 检查消息是否为流式传输
		if msg.IsStreaming {
			// 设置消息流为自动关闭模式
			msg.MessageStream.SetAutomaticClose()

			// 初始化工具收集器
			var toolCalls []schema.ToolCall

			// 循环接收流中的每一帧数据
			for {
				// 从流中接收一帧数据
				frame, err := msg.MessageStream.Recv()

				// 表示流已完全接收，正常退出循环
				if errors.Is(err, io.EOF) {
					break
				}

				// 返回错误
				if err != nil {
					return "", err
				}

				// 检查帧数据是否有效
				if frame != nil {
					// AI正在生成的文字内容
					if frame.Content != "" {
						// 将帧内容添加到字符串构建器中
						builder.WriteString(frame.Content)
						// 同时将内容实时打印到标准输出（不换行）
						_, _ = fmt.Fprint(os.Stdout, frame.Content)
					}
					// 收集AI想要调用的工具
					if len(frame.ToolCalls) > 0 {
						toolCalls = append(toolCalls, frame.ToolCalls...)
					}
				}
			}
			// 打印工具调用信息
			for _, toolCall := range toolCalls {
				if toolCall.Function.Name != "" && toolCall.Function.Arguments != "" {
					fmt.Printf("\n[调用工具] %s(%s)\n", toolCall.Function.Name, toolCall.Function.Arguments)
				}
			}
			// 流处理完毕后，打印一个换行符
			_, _ = fmt.Fprintln(os.Stdout)
			continue
		}

		// 处理非流式消息（一次性返回的完整消息）
		if msg.Message != nil {
			// 将消息内容添加到字符串构建器中
			builder.WriteString(msg.Message.Content)
			// 将消息内容打印到标准输出（带换行符）
			_, _ = fmt.Fprintln(os.Stdout, msg.Message.Content)
		} else {
			// 如果消息为空，仅打印一个空行
			_, _ = fmt.Fprintln(os.Stdout)
		}
	}

	// 返回累积的所有助手回复内容
	return builder.String(), nil
}

// getToolMsg 从流式消息变量中提取完整的工具结果字符串
func getToolMsg(msg *adk.MessageVariant) string {
	if msg.IsStreaming && msg.MessageStream != nil {
		var builder strings.Builder
		for {
			chunk, err := msg.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				builder.WriteString(chunk.Content)
			}
		}
		return builder.String()
	}
	if msg.Message != nil {
		return msg.Message.Content
	}
	return ""
}

// truncate 辅助函数：截断超长字符串，用于控制台预览
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// 尝试压缩 JSON 格式后再截断
	var result bytes.Buffer
	if err := json.Compact(&result, []byte(s)); err == nil {
		s = result.String()
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}