package graphTool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
	"github.com/cloudwego/eino-examples/compose/batch/batch"
)

// Input 是工具调用的参数结构体
// 其 JSON 标签被 utils.InferTool 用于自动生成工具的参数 Schema
type Input struct {
	FilePath string `json:"file_path" jsonschema:"description=已上传文档文件的绝对路径"`
	Question string `json:"question"  jsonschema:"description=需要根据文档回答的问题"`
}

// Output 是工具返回的结构化结果。
type Output struct {
	Answer  string   `json:"answer"`
	Sources []string `json:"sources"` // 用于生成答案的关键摘录来源
}

// scoreTask 是喂给内部 BatchNode 工作流的单个分块输入
type scoreTask struct {
	Text     string
	Question string
}

// scoredChunk 是内部 BatchNode 工作流生成的单个分块结果
type scoredChunk struct {
	Text    string
	Score   int    // 0–10 分，表示与问题的相关性
	Excerpt string // 该分块中最相关的句子或短语
}

// scoreIn 是外部 "score" Lambda 节点的输入。
// 它通过字段映射从两个来源汇集而成：
//   - Chunks: "chunk" 节点的完整输出 ([]*schema.Document)
//   - Question: START 节点的 Question 字段 (来自 Input)
type scoreIn struct {
	Chunks   []*schema.Document
	Question string
}

// synthIn 是 "synthesize" Lambda 节点的输入
// 它通过字段映射从两个来源汇集而成：
//   - TopK: "filter" 节点的完整输出 ([]scoredChunk)
//   - Question: START 节点的 Question 字段 (来自 Input)
type synthIn struct {
	TopK     []scoredChunk
	Question string
}

// BuildTool 构建由 RAG 工作流支持的 answer_from_document 工具
// 它使用了 graphtool.NewInvokableGraphTool，该工具在每次调用时编译工作流 并支持通过内置的检查点存储（checkpoint store）进行中断/恢复
func BuildTool(ctx context.Context, cm model.BaseChatModel) (tool.BaseTool, error) {
	wf := buildWorkflow(cm)
	return graphtool.NewInvokableGraphTool[Input, Output](
		wf,
		"answer_from_document",
		"在大型上传文档中搜索与问题相关的内容，并从最相关的段落中合成带有引用的答案。"+
			"当文档太大无法放入上下文窗口时，请使用此工具代替 read_file。",
	)
}

// buildWorkflow 构建 RAG compose.Workflow（未编译状态）
// graphtool.NewInvokableGraphTool 会在每次调用时对其进行编译
func buildWorkflow(cm model.BaseChatModel) *compose.Workflow[Input, Output] {
	scoreWF := newScoreWorkflow(cm)
	scorer := batch.NewBatchNode(&batch.NodeConfig[scoreTask, scoredChunk]{
		Name:           "ChunkScorer",
		InnerTask:      scoreWF,
		MaxConcurrency: 5,
	})

	wf := compose.NewWorkflow[Input, Output]()

	// load: 从磁盘读取文件，输出单个 Document
	wf.AddLambdaNode("load", compose.InvokableLambda(
		func(ctx context.Context, in Input) ([]*schema.Document, error) {
			data, err := os.ReadFile(in.FilePath)
			if err != nil {
				return nil, fmt.Errorf("read %q: %w", in.FilePath, err)
			}
			return []*schema.Document{{Content: string(data)}}, nil
		},
	)).AddInput(compose.START)

	// chunk: 将每个 Document 拆分成约 800 字符的片段
	wf.AddLambdaNode("chunk", compose.InvokableLambda(
		func(ctx context.Context, docs []*schema.Document) ([]*schema.Document, error) {
			var out []*schema.Document
			for _, d := range docs {
				out = append(out, splitIntoChunks(d.Content, 800)...)
			}
			return out, nil
		},
	)).AddInput("load")

	// score: 通过 BatchNode 并行针对问题对每个分块进行评分
	// Chunks 来自 "chunk" 节点；Question 直接来自 START 节点
	// 两者都使用 WithNoDirectDependency，因为执行顺序已经由
	// START→load→chunk→score 的直接边缘建立
	wf.AddLambdaNode("score", compose.InvokableLambda(
		func(ctx context.Context, in map[string]any) ([]scoredChunk, error) {
        // 从 map 中手动提取字段，并进行类型断言
        // 注意：这里的 key 必须和下面 AddInputWithOptions 里的字段名完全一致
        chunks, ok1 := in["Chunks"].([]*schema.Document)
        question, ok2 := in["Question"].(string)
        
        if !ok1 || !ok2 {
            return nil, fmt.Errorf("score节点输入类型错误: Chunks 类型断言成功=%v, Question 类型断言成功=%v", ok1, ok2)
        }
        tasks := make([]scoreTask, len(chunks))
        for i, c := range chunks {
            tasks[i] = scoreTask{Text: c.Content, Question: question}
        }
        return scorer.Invoke(ctx, tasks)
    },
	)).
		AddInputWithOptions("chunk",
			[]*compose.FieldMapping{compose.ToField("Chunks")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Question", "Question")},
			compose.WithNoDirectDependency())

	// filter: 按分数降序排列，保留最多 3 个分数 ≥ 3 的分块
	wf.AddLambdaNode("filter", compose.InvokableLambda(
		func(ctx context.Context, scored []scoredChunk) ([]scoredChunk, error) {
			sort.Slice(scored, func(i, j int) bool {
				return scored[i].Score > scored[j].Score
			})
			const maxK = 3
			var top []scoredChunk
			for _, c := range scored {
				if c.Score < 3 {
					break
				}
				top = append(top, c)
				if len(top) == maxK {
					break
				}
			}
			return top, nil
		},
	)).AddInput("score")

	// answer: 从前 K 个分块中合成回答，如果为空则返回未找到消息
	// TopK 来自 "filter" 节点；Question 直接来自 START 节点
	// 两者都使用 WithNoDirectDependency："filter" 节点通过其直接边缘控制执行顺序
	wf.AddLambdaNode("answer", compose.InvokableLambda(
		func(ctx context.Context, in map[string]any) (Output, error) {
        topK, ok1 := in["TopK"].([]scoredChunk)
        question, ok2 := in["Question"].(string)
        if !ok1 || !ok2 {
            return Output{}, fmt.Errorf("answer node input error: topK or question missing")
        }
        if len(topK) == 0 {
            return Output{
                Answer: fmt.Sprintf("在文档中未找到与 %q 相关的内容", question),
            }, nil
        }
        
        // 构造临时结构体传给 synthesize 函数，或者直接修改 synthesize 接收 map
        return synthesize(ctx, cm, synthIn{TopK: topK, Question: question})
    },
	)).
		AddInputWithOptions("filter",
			[]*compose.FieldMapping{compose.ToField("TopK")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Question", "Question")},
			compose.WithNoDirectDependency())

	// END 节点接收来自 answer 的输出
	wf.End().
		AddInput("answer")

	return wf
}

// newScoreWorkflow 构建用于每个 BatchNode 任务的单节点内部工作流
func newScoreWorkflow(cm model.BaseChatModel) *compose.Workflow[scoreTask, scoredChunk] {
	wf := compose.NewWorkflow[scoreTask, scoredChunk]()
	wf.AddLambdaNode("score_chunk", compose.InvokableLambda(
		func(ctx context.Context, t scoreTask) (scoredChunk, error) {
			return scoreOneChunk(ctx, cm, t)
		},
	)).AddInput(compose.START)
	wf.End().AddInput("score_chunk")
	return wf
}

// scoreOneChunk 要求模型对单个分块的相关性进行评分（0–10）并提取最相关的摘录
// 解析错误会被视为 0 分，因此错误的 JSON 响应永远不会导致整个流水线中止
func scoreOneChunk(ctx context.Context, cm model.BaseChatModel, t scoreTask) (scoredChunk, error) {
	prompt := fmt.Sprintf(`请评分以下文本分块与问题的相关性。

问题: %s

文本分块:
%s

仅回复 JSON — 不要解释，不要 Markdown 代码块：
{"score": <0-10>, "excerpt": "<最相关的句子或短语，如果分数位 0 则为空字符串>"}

评分指南: 0=完全无关, 3=略微相关, 7=清晰相关, 10=直接回答了问题。`,
		t.Question, t.Text)

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		// 将模型错误视为不相关，而不是中止整个批处理
		return scoredChunk{Text: t.Text, Score: 0}, nil
	}

	content := strings.TrimSpace(resp.Content)
	// 去除可选的 markdown 代码块包装
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var sr struct {
		Score   int    `json:"score"`
		Excerpt string `json:"excerpt"`
	}
	if err := json.Unmarshal([]byte(content), &sr); err != nil {
		return scoredChunk{Text: t.Text, Score: 0}, nil
	}
	return scoredChunk{Text: t.Text, Score: sr.Score, Excerpt: sr.Excerpt}, nil
}

// synthesize 根据前 K 个分块构建提示词并生成带有引用的回答。
func synthesize(ctx context.Context, cm model.BaseChatModel, in synthIn) (Output, error) {
	var sb strings.Builder
	sb.WriteString("仅使用提供的文档摘录回答以下问题。\n\n")
	sb.WriteString("问题: ")
	sb.WriteString(in.Question)
	sb.WriteString("\n\n文档摘录:\n")

	sources := make([]string, len(in.TopK))
	for i, c := range in.TopK {
		excerpt := c.Excerpt
		if excerpt == "" {
			excerpt = c.Text
		}
		sources[i] = excerpt
		fmt.Fprintf(&sb, "[%d] %s\n\n", i+1, excerpt)
	}
	sb.WriteString("提供简洁清晰的回答。在引用来源时，请注明摘录编号，如 [1]。")

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(sb.String())})
	if err != nil {
		return Output{}, fmt.Errorf("synthesize: %w", err)
	}
	return Output{Answer: resp.Content, Sources: sources}, nil
}

// splitIntoChunks 将文本切分为最大 chunkSize 字符的分块，
// 尽可能在段落边界 (\n\n) 处断开，其次是换行符。
func splitIntoChunks(text string, chunkSize int) []*schema.Document {
	var chunks []*schema.Document
	var buf strings.Builder

	flush := func() {
		s := strings.TrimSpace(buf.String())
		if s != "" {
			chunks = append(chunks, &schema.Document{Content: s})
		}
		buf.Reset()
	}

	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if buf.Len()+len(para)+2 > chunkSize && buf.Len() > 0 {
			flush()
		}
		// 如果段落本身超过了 chunkSize：按行拆分
		if len(para) > chunkSize {
			for _, line := range strings.Split(para, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				if buf.Len()+len(line)+1 > chunkSize && buf.Len() > 0 {
					flush()
				}
				if buf.Len() > 0 {
					buf.WriteByte('\n')
				}
				buf.WriteString(line)
			}
		} else {
			if buf.Len() > 0 {
				buf.WriteString("\n\n")
			}
			buf.WriteString(para)
		}
	}
	flush()
	return chunks
}

// 该包提供了一个由 compose.Workflow 支持的 answer_from_document（根据文档回答问题）工具。
//
// 该工作流使用字段映射（field mapping）在非相邻节点（score, answer）之间共享用户的提问，
// 而无需将提问参数穿透传递给每一个中间节点的输出类型：
//
//	开始节点 START{文件路径 FilePath, 提问 Question}
//	   │ (通过 WithNoDirectDependency 传递数据)─────────────────────────────────────┐
//	  ▼                                                                            │ 提问 Question
//	[load 节点]  os.ReadFile → []*schema.Document (读取文档)                       │
//	  ▼                                                                            │
//	[chunk 节点] 段落拆分器 → []*schema.Document (分块)                            │
//	  ▼  分块 Chunks ──────────────────────────────────────────────────────────► [score 节点]
//	                                                                                │ []scoredChunk (已评分分块)
//	                                                                               ▼
//	                                                                            [filter 节点] 获取 Top-K
//	                                                                                │ TopK (可能为空)
//	                                                                               ▼
//	                                                                            [answer 节点] ◄─ 提问 (来自 START)
//	                                                                     (在线合成答案或返回未找到)
//	                                                                                │
//	                                                                              结束 END
//
// [score] 节点包装了一个 BatchNode，其内部工作流通过并行调用 ChatModel 对每个分块进行评分（最大并发数为 5）。