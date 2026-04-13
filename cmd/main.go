package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/joho/godotenv"
	"github.com/cloudwego/eino/schema"
)

const AmiyaSystemPrompt = `你是阿米娅（Amiya），罗德岛制药公司的领导人之一，也是博士最信赖的助手。

## 你的身份背景
- 你是一名卡特斯（Cautus，兔子亚人种族），拥有兔耳和柔软的性格
- 你是罗德岛的公开领袖，负责指挥行动、协调各部门
- 你深深信任、爱慕、尊崇并依赖"博士"（就是正在和你说话的人）
- 你经历过许多艰难的战斗，但依然保持善良和希望

## 你的性格特点
- 温柔、认真、有责任感
- 偶尔会露出一点小紧张，但总是努力做到最好
- 对博士非常关心，会提醒博士注意休息
- 说话方式温和有礼，会用"博士"来称呼对方
- 在专业问题上会认真分析，给出可靠的建议

## 说话风格
- 常用语气词："嗯"、"那个"、"请"
- 经常说"博士，请多休息"之类的关心话
- 汇报工作时会很专业，但日常对话时会展现可爱的一面
- 不会过度卖萌，保持一定的成熟感

请以阿米娅的身份来回应博士的话。保持角色一致，不要跳出角色。`

func main() {
	// 初始化上下文
	ctx := context.Background()

	// 获取apiKey等环境变量并封装为config
	err := godotenv.Load()
    if err != nil {
        log.Println("未找到 .env 文件，使用系统环境变量")
    }
	apiKey,baseURL,modelName := os.Getenv("OPENAI_API_KEY"),os.Getenv("OPENAI_BASE_URL"),os.Getenv("OPENAI_MODEL_NAME")
	if apiKey == "" || baseURL == "" || modelName == "" {
		log.Fatal("请设置环境变量")
	}

	modelConfig := &openai.ChatModelConfig{
		Model:  modelName,
		APIKey: apiKey,
		BaseURL: baseURL,
	}

	// 创建 ChatModel 实例
	chatModel, err := openai.NewChatModel(ctx,modelConfig)
	if err != nil {
		log.Fatal("创建 ChatModel 实例失败:", err)
	}

	// 初始化对话历史和系统消息
	messages := []*schema.Message{
		schema.SystemMessage(AmiyaSystemPrompt),
	}

	fmt.Println("====================================")
	fmt.Println("  阿米娅终端")
	fmt.Println("  输入 'exit' 退出")
	fmt.Println("====================================")
	fmt.Println()
	fmt.Println("阿米娅: 博士，您好！我是阿米娅，有什么需要我帮忙的吗？")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for{
		fmt.Print("博士：")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" {
			fmt.Println("\n阿米娅: 博士，注意休息哦，我会一直在这里等您的。再见！")
			break
		}

		// 将对话加入对话历史
		messages = append(messages, schema.UserMessage(input))

		// 调用 ChatModel
		resp, err := chatModel.Generate(ctx, messages)
		if err != nil {
			log.Println("生成回复失败:", err)
			// 回滚消息历史
			messages = messages[:len(messages)-1]
			continue
		}

		content := resp.Content
		messages = append(messages, resp)

		fmt.Printf("\n阿米娅: %s\n\n", content)
	}
}