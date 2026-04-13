package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

const (
	AmiyaName = "Amiya"
	
	AmiyaDescription = "明日方舟世界观里罗德岛制药公司的领导人，博士最信赖的助手。一名温柔认真的卡特斯(兔子亚人种族)，负责指挥行动、协调各部门，并可以查询干员信息、查看资源状况、制定作战计划。"
	
	AmiyaInstruction = `你是阿米娅（Amiya），明日方舟世界观里罗德岛制药公司的领导人之一，也是博士最信赖的助手。

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
- 不会过度卖萌，保持一定的成熟感`
)

// type OperatorQueryInput struct{
// 	Name string `json:"name" jsonschema:"description=干员名称，如'能天使'、'银灰'等"`
// }

// func queryOperator(ctx context.Context, input *OperatorQueryInput) (string, error){
// 	// 模拟干员数据库
// 	operators := map[string]string{
// 		"能天使": "【能天使】★★★★★★ | 职业：狙击 | 特长：对空攻击，高射速 | 天赋：攻击速度提升",
// 		"银灰":  "【银灰】★★★★★★ | 职业：近卫 | 特长：远程斩击 | 天赋：再部署时间缩短",
// 		"陈":   "【陈】★★★★★★ | 职业：近卫 | 特长：多段爆发攻击 | 天赋：技力恢复速度提升",
// 		"艾雅法拉": "【艾雅法拉】★★★★★★ | 职业：术师 | 特长：群体法术伤害 | 天赋：法术伤害提升",
// 		"阿米娅": "【阿米娅】★★★★★★ | 职业：术师/近卫 | 特长：真实伤害 | 天赋：每次攻击回复技力",
// 		"德克萨斯": "【德克萨斯】★★★★★ | 职业：先锋 | 特长：产生费用，群体眩晕 | 天赋：初始Cost+1",
// 		"蓝毒":  "【蓝毒】★★★★★ | 职业：狙击 | 特长：多目标攻击，法术dot伤害 | 天赋：攻击附带法术伤害",
// 	}

// 	info, ofk := operators[input.Name]
// 	if !ofk {
// 		return fmt.Sprintf("未找到名为'%s'的干员记录", input.Name), nil
// 	}

// 	return info, nil
// }

// type ResourceQueryInpout struct{
// 	ResourceType string `json:"resource_type" jsonschema:"description=资源类型，可选值：龙门币、合成玉、理智、源石、全部"`
// }

// func queryResources(ctx context.Context, input *ResourceQueryInpout) (string, error){
// 	resources := map[string]string{
// 		"龙门币": "龙门币: 1,234,567",
// 		"合成玉": "合成玉: 8,900",
// 		"理智":  "理智: 130/135（将在 2 小时后回满）",
// 		"源石":  "源石: 42 个",
// 	}

// 	if input.ResourceType == "全部" || input.ResourceType == "" {
// 		var result strings.Builder
// 		result.WriteString("=== 罗德岛资源报告 ===\n")
// 		for _, v := range resources {
// 			result.WriteString("  " + v + "\n")
// 		}
// 		result.WriteString("报告时间：刚刚更新")
// 		return result.String(), nil
// 	}

// 	info, ok := resources[input.ResourceType]
// 	if !ok {
// 		return fmt.Sprintf("未知的资源类型: '%s'。支持查询：龙门币、合成玉、理智、源石、全部", input.ResourceType), nil
// 	}

// 	return info, nil
// }

// type BattlePlanInput struct {
// 	StageName  string `json:"stage_name" jsonschema:"description=关卡名称，如'1-7'、'CE-5'、'SK-5'等"`
// 	Difficulty string `json:"difficulty" jsonschema:"description=难度偏好，可选值：标准、挑战"`
// }

// func makeBattlePlan(ctx context.Context, input *BattlePlanInput) (string, error) {
// 	plan := fmt.Sprintf(`=== 作战计划：%s（%s模式）===
// 推荐编队：
//   先锋位：德克萨斯 — 快速获取部署费用
//   狙击位：能天使 — 对空防御 + 高DPS
//   近卫位：银灰 — 核心输出，关键时刻真银斩
//   术师位：艾雅法拉 — 群体法术清场
//   医疗位：闪灵 — 全队治疗保障
//   重装位：星熊 — 前排抗压

// 作战要点：
//   1. 开局立即部署德克萨斯获取初始费用
//   2. 优先在高台部署能天使控制空中威胁
//   3. 银灰部署在关键路口，技能 3 留到敌方精英出现时使用
//   4. 艾雅法拉在敌方密集时释放火山，最大化群伤

// 预估理智消耗：%d
// 预估通关概率：%s`, input.StageName, input.Difficulty, 30, "95%")

// 	return plan, nil
// }

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


	// 创建 ChatModel 实例
	modelConfig := &openai.ChatModelConfig{
		Model:  modelName,
		APIKey: apiKey,
		BaseURL: baseURL,
	}

	chatModel, err := openai.NewChatModel(ctx,modelConfig)
	if err != nil {
		log.Fatal("创建 ChatModel 实例失败:", err)
	}
	log.Println("ChatModel 实例创建成功")

	// 创建Agent实例
	agentConfig := &adk.ChatModelAgentConfig{
		Name: AmiyaName,
		Description: AmiyaDescription,
		Instruction: AmiyaInstruction,
		Model: chatModel,
	}

	agent,err := adk.NewChatModelAgent(ctx,agentConfig)
	if err != nil {
		log.Fatal("创建 ChatModelAgent 实例失败:", err)
	}

	// 创建agentRunner实例
	runner := adk.NewRunner(ctx,adk.RunnerConfig{
		Agent: agent,
		EnableStreaming: true,
	})
	
	// 初始化对话历史和系统消息（Eino会自动把Agent的Instruction注入为System Message）
	messages := make([]*schema.Message, 0, 16)

	// 启动对话
	fmt.Println()
	fmt.Println("===============================================开始对话=======================================================")
	fmt.Println()
	fmt.Println("阿米娅: 博士，您好！我是阿米娅，有什么需要我帮忙的吗？")
	fmt.Println()

	//log.Println("")

	scanner := bufio.NewScanner(os.Stdin)
	for{
		fmt.Print("博士：")
		if !scanner.Scan() {
			break
		}

		fmt.Println()

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// 将用户消息加入对话历史
		messages = append(messages, schema.UserMessage(input))

		// 调用 AgentRunner 生成回复
		events := runner.Run(ctx, messages)
		content,err := getAssistantMsg(events)
		if err != nil {
			log.Println("生成回复失败:", err)
			// 回滚消息历史
			messages = messages[:len(messages)-1]
			continue
		}

		// 将模型回复加入对话历史
		messages = append(messages, schema.AssistantMessage(content,nil))

		fmt.Println()
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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

		// 只处理Assistant消息
		if msg.Role != schema.Assistant {
			continue
		}

		// 检查消息是否为流式传输
		if msg.IsStreaming {
			// 设置消息流为自动关闭模式
			msg.MessageStream.SetAutomaticClose()

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

				// 检查帧数据是否有效且包含内容
				if frame != nil && frame.Content != "" {
					// 将帧内容添加到字符串构建器中
					builder.WriteString(frame.Content)
					// 同时将内容实时打印到标准输出（不换行）
					_, _ = fmt.Fprint(os.Stdout, frame.Content)
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