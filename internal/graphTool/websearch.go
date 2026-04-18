package graphTool

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
	"github.com/cloudwego/eino-examples/compose/batch/batch"
)

type WebSearchInput struct {
	Query string `json:"query" jsonschema:"description=需要搜索的问题或关键词"`
}

type WebSearchOutput struct {
	Answer     string   `json:"answer"`
	Sources    []string `json:"sources"`
	SearchTime string   `json:"search_time"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

type fetchTask struct {
	URL     string
	Title   string
	Snippet string
	Query   string
}

type fetchedPage struct {
	URL       string
	Title     string
	Snippet   string
	Content   string
	Truncated bool
}

type cleanIn struct {
	Pages []fetchedPage
	Query string
}

type scoredPage struct {
	fetchedPage
	Score int
}

func BuildWebSearchTool(ctx context.Context, cm model.BaseChatModel) (tool.BaseTool, error) {
	wf := buildWebSearchWorkflow(cm)
	return graphtool.NewInvokableGraphTool[WebSearchInput, WebSearchOutput](
		wf,
		"web_search",
		"联网搜索工具。当需要查询实时信息、新闻、最新数据或互联网上的内容时使用此工具。"+
			"会自动搜索、抓取网页并总结关键信息。",
	)
}

func buildWebSearchWorkflow(cm model.BaseChatModel) *compose.Workflow[WebSearchInput, WebSearchOutput] {
	fetchWF := newFetchWorkflow()
	fetcher := batch.NewBatchNode(&batch.NodeConfig[fetchTask, fetchedPage]{
		Name:           "PageFetcher",
		InnerTask:      fetchWF,
		MaxConcurrency: 3,
	})

	wf := compose.NewWorkflow[WebSearchInput, WebSearchOutput]()

	wf.AddLambdaNode("search", compose.InvokableLambda(
		func(ctx context.Context, in WebSearchInput) ([]SearchResult, error) {
			return performSearch(ctx, in.Query)
		},
	)).AddInput(compose.START)

	wf.AddLambdaNode("select_urls", compose.InvokableLambda(
		func(ctx context.Context, results []SearchResult) ([]fetchTask, error) {
			maxResults := 5
			if len(results) > maxResults {
				results = results[:maxResults]
			}
			tasks := make([]fetchTask, len(results))
			for i, r := range results {
				tasks[i] = fetchTask{
					URL:     r.Link,
					Title:   r.Title,
					Snippet: r.Snippet,
				}
			}
			return tasks, nil
		},
	)).AddInput("search")

	wf.AddLambdaNode("fetch", compose.InvokableLambda(
		func(ctx context.Context, in map[string]any) ([]fetchedPage, error) {
			tasks, ok1 := in["Tasks"].([]fetchTask)
			query, ok2 := in["Query"].(string)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("fetch节点输入类型错误")
			}
			for i := range tasks {
				tasks[i].Query = query
			}
			return fetcher.Invoke(ctx, tasks)
		},
	)).
		AddInputWithOptions("select_urls",
			[]*compose.FieldMapping{compose.ToField("Tasks")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Query", "Query")},
			compose.WithNoDirectDependency())

	wf.AddLambdaNode("clean", compose.InvokableLambda(
		func(ctx context.Context, in map[string]any) ([]scoredPage, error) {
			pages, ok1 := in["Pages"].([]fetchedPage)
			query, ok2 := in["Query"].(string)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("clean节点输入类型错误")
			}
			return cleanAndScorePages(ctx, cm, cleanIn{Pages: pages, Query: query})
		},
	)).
		AddInputWithOptions("fetch",
			[]*compose.FieldMapping{compose.ToField("Pages")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Query", "Query")},
			compose.WithNoDirectDependency())

	wf.AddLambdaNode("summarize", compose.InvokableLambda(
		func(ctx context.Context, in map[string]any) (WebSearchOutput, error) {
			scoredPages, ok1 := in["ScoredPages"].([]scoredPage)
			query, ok2 := in["Query"].(string)
			if !ok1 || !ok2 {
				return WebSearchOutput{}, fmt.Errorf("summarize节点输入类型错误")
			}
			return synthesizeWebAnswer(ctx, cm, scoredPages, query)
		},
	)).
		AddInputWithOptions("clean",
			[]*compose.FieldMapping{compose.ToField("ScoredPages")},
			compose.WithNoDirectDependency()).
		AddInputWithOptions(compose.START,
			[]*compose.FieldMapping{compose.MapFields("Query", "Query")},
			compose.WithNoDirectDependency())

	wf.End().AddInput("summarize")

	return wf
}

func newFetchWorkflow() *compose.Workflow[fetchTask, fetchedPage] {
	wf := compose.NewWorkflow[fetchTask, fetchedPage]()
	wf.AddLambdaNode("fetch_page", compose.InvokableLambda(
		func(ctx context.Context, t fetchTask) (fetchedPage, error) {
			return fetchWebPage(ctx, t)
		},
	)).AddInput(compose.START)
	wf.End().AddInput("fetch_page")
	return wf
}

func performSearch(ctx context.Context, query string) ([]SearchResult, error) {
	log.Printf("[web_search] 开始搜索: %s", query)

	results, err := searchDuckDuckGoHTML(ctx, query)
	if err != nil {
		log.Printf("[web_search] DuckDuckGo 搜索失败: %v", err)
	} else if len(results) > 0 {
		log.Printf("[web_search] DuckDuckGo 返回 %d 条结果", len(results))
		return results, nil
	}

	results, err = searchSearxNG(ctx, query)
	if err != nil {
		log.Printf("[web_search] SearxNG 搜索失败: %v", err)
	} else if len(results) > 0 {
		log.Printf("[web_search] SearxNG 返回 %d 条结果", len(results))
		return results, nil
	}

	results, err = searchWikipedia(ctx, query)
	if err != nil {
		log.Printf("[web_search] Wikipedia 搜索失败: %v", err)
	} else if len(results) > 0 {
		log.Printf("[web_search] Wikipedia 返回 %d 条结果", len(results))
		return results, nil
	}

	log.Printf("[web_search] 所有搜索源均未返回结果")
	return nil, fmt.Errorf("未找到相关搜索结果")
}

func searchWikipedia(ctx context.Context, query string) ([]SearchResult, error) {
	log.Printf("[web_search] 开始 Wikipedia 搜索: %s", query)
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://zh.wikipedia.org/w/api.php?action=opensearch&search=%s&limit=5&format=json", encodedQuery)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[web_search] Wikipedia 请求失败: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[web_search] Wikipedia 响应状态码: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Printf("[web_search] Wikipedia 响应长度: %d 字节", len(body))

	var wikiResp []json.RawMessage
	if err := json.Unmarshal(body, &wikiResp); err != nil {
		return nil, err
	}

	if len(wikiResp) < 4 {
		return nil, fmt.Errorf("Wikipedia 响应格式错误")
	}

	var titles []string
	var descriptions []string
	var urls []string

	if err := json.Unmarshal(wikiResp[1], &titles); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(wikiResp[2], &descriptions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(wikiResp[3], &urls); err != nil {
		return nil, err
	}

	var results []SearchResult
	for i := range titles {
		if i < len(descriptions) && i < len(urls) {
			results = append(results, SearchResult{
				Title:   titles[i],
				Link:    urls[i],
				Snippet: descriptions[i],
			})
		}
	}

	log.Printf("[web_search] Wikipedia 返回 %d 条结果", len(results))

	return results, nil
}

func searchDuckDuckGoHTML(ctx context.Context, query string) ([]SearchResult, error) {
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", encodedQuery)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建搜索请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "identity")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[web_search] DuckDuckGo 响应状态码: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("搜索返回错误状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取搜索结果失败: %w", err)
	}

	html := string(body)
	log.Printf("[web_search] DuckDuckGo 响应长度: %d 字节", len(html))

	results := parseDuckDuckGoHTML(html)
	log.Printf("[web_search] DuckDuckGo 解析出 %d 条结果", len(results))

	return results, nil
}

func parseDuckDuckGoHTML(html string) []SearchResult {
	var results []SearchResult

	resultRegex := regexp.MustCompile(`<a[^>]*class="[^"]*result__a[^"]*"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
	snippetRegex := regexp.MustCompile(`<a[^>]*class="[^"]*result__snippet[^"]*"[^>]*>([^<]*(?:<[^>]+>[^<]*)*)</a>`)

	resultMatches := resultRegex.FindAllStringSubmatch(html, -1)
	snippetMatches := snippetRegex.FindAllStringSubmatch(html, -1)

	log.Printf("[web_search] 正则匹配到 %d 个结果链接", len(resultMatches))

	for i, match := range resultMatches {
		if len(match) >= 3 {
			link := match[1]
			title := cleanHTMLTags(match[2])

			if strings.Contains(link, "y.js") {
				continue
			}

			if strings.HasPrefix(link, "//") {
				link = "https:" + link
			}

			if uddgIdx := strings.Index(link, "uddg="); uddgIdx != -1 {
				encodedURL := link[uddgIdx+5:]
				if ampIdx := strings.Index(encodedURL, "&"); ampIdx != -1 {
					encodedURL = encodedURL[:ampIdx]
				}
				if decoded, err := url.QueryUnescape(encodedURL); err == nil {
					link = decoded
				}
			} else if strings.Contains(link, "duckduckgo.com/l/?") {
				if decoded, err := url.QueryUnescape(link); err == nil {
					if uddgIdx := strings.Index(decoded, "uddg="); uddgIdx != -1 {
						encodedURL := decoded[uddgIdx+5:]
						if ampIdx := strings.Index(encodedURL, "&"); ampIdx != -1 {
							encodedURL = encodedURL[:ampIdx]
						}
						if finalURL, err := url.QueryUnescape(encodedURL); err == nil {
							link = finalURL
						}
					}
				}
			}

			if title != "" && link != "" && !strings.Contains(link, "duckduckgo.com") {
				results = append(results, SearchResult{
					Title:   strings.TrimSpace(title),
					Link:    strings.TrimSpace(link),
					Snippet: strings.TrimSpace(getSnippet(snippetMatches, i)),
				})
				log.Printf("[web_search] 解析结果: %s -> %s", title, link)
			}
		}

		if len(results) >= 8 {
			break
		}
	}

	return results
}

func getSnippet(snippetMatches [][]string, index int) string {
	if index < len(snippetMatches) && len(snippetMatches[index]) >= 2 {
		return cleanHTMLTags(snippetMatches[index][1])
	}
	return ""
}

func searchSearxNG(ctx context.Context, query string) ([]SearchResult, error) {
	encodedQuery := url.QueryEscape(query)
	searchURL := fmt.Sprintf("https://searx.be/search?q=%s&format=json", encodedQuery)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searxResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searxResp); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, r := range searxResp.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			Link:    r.URL,
			Snippet: r.Content,
		})
		if len(results) >= 8 {
			break
		}
	}

	return results, nil
}

func cleanHTMLTags(s string) string {
	tagRegex := regexp.MustCompile(`<[^>]+>`)
	s = tagRegex.ReplaceAllString(s, "")
	entityRegex := regexp.MustCompile(`&[^;]+;`)
	s = entityRegex.ReplaceAllStringFunc(s, func(e string) string {
		switch e {
		case "&amp;":
			return "&"
		case "&lt;":
			return "<"
		case "&gt;":
			return ">"
		case "&quot;":
			return "\""
		case "&#39;":
			return "'"
		default:
			return " "
		}
	})
	return strings.TrimSpace(s)
}

func fetchWebPage(ctx context.Context, t fetchTask) (fetchedPage, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", t.URL, nil)
	if err != nil {
		return fetchedPage{URL: t.URL, Title: t.Title, Snippet: t.Snippet, Content: "", Truncated: false}, nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return fetchedPage{URL: t.URL, Title: t.Title, Snippet: t.Snippet, Content: "", Truncated: false}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fetchedPage{URL: t.URL, Title: t.Title, Snippet: t.Snippet, Content: "", Truncated: false}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fetchedPage{URL: t.URL, Title: t.Title, Snippet: t.Snippet, Content: "", Truncated: false}, nil
	}

	content := string(body)
	content = stripHTMLTags(content)
	content = cleanText(content)

	maxContentLen := 8000
	truncated := false
	if len(content) > maxContentLen {
		content = content[:maxContentLen]
		truncated = true
	}

	return fetchedPage{
		URL:       t.URL,
		Title:     t.Title,
		Snippet:   t.Snippet,
		Content:   content,
		Truncated: truncated,
	}, nil
}

func stripHTMLTags(html string) string {
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	styleRegex := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	html = scriptRegex.ReplaceAllString(html, "")
	html = styleRegex.ReplaceAllString(html, "")

	tagRegex := regexp.MustCompile(`<[^>]+>`)
	text := tagRegex.ReplaceAllString(html, " ")

	entityRegex := regexp.MustCompile(`&[^;]+;`)
	text = entityRegex.ReplaceAllString(text, " ")

	return text
}

func cleanText(text string) string {
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\r", "")

	spaceRegex := regexp.MustCompile(` {2,}`)
	text = spaceRegex.ReplaceAllString(text, " ")

	newlineRegex := regexp.MustCompile(`\n{3,}`)
	text = newlineRegex.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

func cleanAndScorePages(ctx context.Context, cm model.BaseChatModel, in cleanIn) ([]scoredPage, error) {
	var scored []scoredPage

	for _, page := range in.Pages {
		if page.Content == "" {
			continue
		}

		score, err := scorePageRelevance(ctx, cm, page, in.Query)
		if err != nil {
			score = 0
		}

		if score >= 3 {
			summary, err := summarizePage(ctx, cm, page, in.Query)
			if err != nil {
				summary = page.Content
				if len(summary) > 1500 {
					summary = summary[:1500]
				}
			}
			page.Content = summary
			scored = append(scored, scoredPage{
				fetchedPage: page,
				Score:       score,
			})
		}
	}

	if len(scored) == 0 {
		for _, page := range in.Pages {
			if page.Content != "" {
				scored = append(scored, scoredPage{
					fetchedPage: page,
					Score:       1,
				})
				if len(scored) >= 3 {
					break
				}
			}
		}
	}

	return scored, nil
}

func scorePageRelevance(ctx context.Context, cm model.BaseChatModel, page fetchedPage, query string) (int, error) {
	content := page.Content
	if len(content) > 2000 {
		content = content[:2000]
	}

	prompt := fmt.Sprintf(`请评估以下网页内容与搜索问题的相关性。

搜索问题: %s

网页标题: %s
网页摘要: %s

网页内容片段:
%s

仅回复一个 0-10 的整数评分，不要其他任何内容：
0 = 完全无关
3 = 略微相关
5 = 有一定相关性
7 = 比较相关
10 = 直接回答了问题`, query, page.Title, page.Snippet, content)

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		return 0, err
	}

	content = strings.TrimSpace(resp.Content)
	var score int
	if _, err := fmt.Sscanf(content, "%d", &score); err != nil {
		return 0, fmt.Errorf("解析评分失败: %w", err)
	}

	if score < 0 {
		score = 0
	}
	if score > 10 {
		score = 10
	}

	return score, nil
}

func summarizePage(ctx context.Context, cm model.BaseChatModel, page fetchedPage, query string) (string, error) {
	content := page.Content
	if len(content) > 4000 {
		content = content[:4000]
	}

	prompt := fmt.Sprintf(`请从以下网页内容中提取与搜索问题相关的关键信息。

搜索问题: %s

网页标题: %s
网页URL: %s

网页内容:
%s

要求：
1. 只提取与问题直接相关的信息
2. 保留关键数据和事实
3. 去除无关内容、广告、导航等
4. 用简洁的语言总结，不超过500字

请直接输出总结内容，不要其他说明：`, query, page.Title, page.URL, content)

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(prompt)})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}

func synthesizeWebAnswer(ctx context.Context, cm model.BaseChatModel, pages []scoredPage, query string) (WebSearchOutput, error) {
	if len(pages) == 0 {
		return WebSearchOutput{
			Answer:     fmt.Sprintf("抱歉，未能找到与「%s」相关的有效信息。请尝试更换关键词或稍后再试。", query),
			Sources:    []string{},
			SearchTime: time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	var sb strings.Builder
	sb.WriteString("请根据以下搜索结果回答问题。\n\n")
	sb.WriteString("问题: ")
	sb.WriteString(query)
	sb.WriteString("\n\n搜索结果:\n")

	sources := make([]string, 0, len(pages))
	for i, page := range pages {
		sb.WriteString(fmt.Sprintf("\n--- 来源 [%d] ---\n", i+1))
		sb.WriteString(fmt.Sprintf("标题: %s\n", page.Title))
		sb.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
		sb.WriteString(fmt.Sprintf("内容摘要:\n%s\n", page.Content))
		sources = append(sources, fmt.Sprintf("%s - %s", page.Title, page.URL))
	}

	sb.WriteString("\n\n请综合以上信息回答问题。要求：\n")
	sb.WriteString("1. 提供准确、有用的回答\n")
	sb.WriteString("2. 在回答中标注信息来源，如 [1]、[2]\n")
	sb.WriteString("3. 如果信息不确定，请明确说明\n")
	sb.WriteString("4. 保持回答简洁清晰")

	resp, err := cm.Generate(ctx, []*schema.Message{schema.UserMessage(sb.String())})
	if err != nil {
		return WebSearchOutput{}, fmt.Errorf("生成回答失败: %w", err)
	}

	return WebSearchOutput{
		Answer:     resp.Content,
		Sources:    sources,
		SearchTime: time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
