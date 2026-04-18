一个用Eino框架实现的阿米娅Agent

运行示例
```
PS D:\CodeStudy\GoProjects\src\AmiyaAgent> go run "d:\CodeStudy\GoProjects\src\AmiyaAgent\cmd\main.go"
2026/04/18 21:50:32 ChatModel 实例创建成功
2026/04/18 21:50:32 CozeLoop追踪已启用
2026/04/18 21:50:32 工具 operator_query 初始化成功
2026/04/18 21:50:32 工具 resource_query 初始化成功
2026/04/18 21:50:32 工具 battle_plan 初始化成功
2026/04/18 21:50:32 工具 rag 初始化成功
2026/04/18 21:50:32 工具 web_search 初始化成功
2026/04/18 21:50:37 DeepAgent 实例创建成功
2026/04/18 21:50:37 AgentRunner 实例创建成功
2026/04/18 21:50:37 会话存储创建成功
请输入会话ID（留空则创建新会话）: 
创建新会话: 7bb20de0-a966-4e79-8f8a-8a29a169fcc0
2026/04/18 21:50:39 会话文件不存在，创建新会话: sessionStore\7bb20de0-a966-4e79-8f8a-8a29a169fcc0.jsonl
2026/04/18 21:50:39 会话获取或创建成功

===============================================开始对话（当前会话标题：新会话）=======================================================

阿米娅: 博士，您好！我是阿米娅，有什么需要我帮忙的吗？

博士：查询干员维什戴尔的信息
阿米娅：2026/04/18 21:50:45 [trace] Agent/Amiya start
2026/04/18 21:50:45 [trace] Agent/Amiya end
2026/04/18 21:50:45 [trace] Lambda/ start
2026/04/18 21:50:45 [trace] Lambda/ end
2026/04/18 21:50:45 [trace] Lambda/Init start
2026/04/18 21:50:45 [trace] Lambda/Init end
2026/04/18 21:50:45 [trace] ChatModel/ChatModel start
博士，我来帮您查询维什戴尔干员的信息。让我先调用相关的查询系统。
[调用工具] skill({"skill":"operator-info"})

2026/04/18 21:50:52 [trace] ToolsNode/ToolNode start
2026/04/18 21:50:52 [trace] ToolsNode/ToolNode error: interrupt signal: ID=9b88e247-46e4-48ba-adfd-c3ab25439edf, Addr=agent:Amiya;runnable:Amiya;node:node_1;node:ToolNode, Info=interrupt info: Info=&{[{0xc000804650 call_42a01bc6650c4e83bf13485d function {skill {"skill":"operator-info"}} map[]}] map[] map[] [call_42a01bc6650c4e83bf13485d] map[call_42a01bc6650c4e83bf13485d:tool 'skill' interrupted with arguments '{"skill":"operator-info"}', waiting for your approval, please answer with Y/N]}, IsRootCause=false, State=interrupt state: State=&{assistant: 博士，我来帮您查询维什戴尔干员的信息。让我先调用相关的查询系统。
reasoning content:
用户要查询干员维什戴尔的信息。这是一个明日方舟相关的查询，我应该使用可用的工具来获取干员信息。

让我看看可用的工具：
1. operator_query - 根据干员名称查询其职业、特长和天赋等基本信息
2. skill工具中有一个operator-info技能 - 查询明日方舟干员的详细信息，包括职业、星级、技能、天赋、基建技能等

用户询问的是干员信息，这符合operator-info技能的描述："当用户询问某个干员的信息时调用此技能。"

根据技能使用说明：
- 当用户请求匹配技能领域时，我必须立即调用此工具
- 这是一个阻塞要求：必须在生成任何其他响应之前调用相关Skill工具
- 使用<name>标签中的精确字符串作为技能名称

所以我应该先调用skill工具来加载operator-info技能。
tool_calls:
index[0]:{Index:0xc000804650 ID:call_42a01bc6650c4e83bf13485d Type:function Function:{Name:skill Arguments:{"skill":"operator-info"}} Extra:map[]}

finish_reason: tool_calls
usage: &{8381 {42} 212 8593 {178}} map[] map[] [call_42a01bc6650c4e83bf13485d]}, LayerSpecificPayload=<nil>, SubsLen=1
2026/04/18 21:50:52 [trace] Graph/ReAct error: interrupt happened, info: &{State:0xc0004860e0 BeforeNodes:[] AfterNodes:[] RerunNodes:[ToolNode] RerunNodesExtra:map[ToolNode:0xc000480af0] SubGraphs:map[] InterruptContexts:[]}
2026/04/18 21:50:52 [trace] Chain/Amiya error: interrupt happened, info: &{State:<nil> BeforeNodes:[] AfterNodes:[] RerunNodes:[] RerunNodesExtra:map[] SubGraphs:map[node_1:0xc000703e00] InterruptContexts:[0xc000481e00]}

⚠️  需要用户审批 ⚠️
工具: skill
参数: {"skill":"operator-info"}

是否批准? (y/n): y
✓ 已批准，正在执行...
2026/04/18 21:50:55 [trace] Agent/Amiya start
2026/04/18 21:50:55 [trace] Agent/Amiya end
2026/04/18 21:50:55 [trace] ToolsNode/ToolNode start
2026/04/18 21:50:55 [trace] Tool/skill start
2026/04/18 21:50:55 [trace] Tool/skill end
[调用工具结果] Launching skill: operator-info
Base directory for this skill: D:\CodeStudy\GoProjects\src\AmiyaAgent\skills\operator-info

# 干员信息查询

## 功能说明

此技能用于查询明日方舟游��...
2026/04/18 21:50:55 [trace] ChatModel/ChatModel start

[调用工具] operator_query({"name":"维什戴尔"})

2026/04/18 21:50:59 [trace] ToolsNode/ToolNode start
2026/04/18 21:50:59 [trace] ToolsNode/ToolNode error: interrupt signal: ID=d9ef28c0-23a1-4c96-893c-d52a13487f85, Addr=agent:Amiya;runnable:Amiya;node:node_1;node:ToolNode, Info=interrupt info: Info=&{[{0xc000298430 call_c0e9aa514886493daba58275 function {operator_query {"name":"维什戴尔"}} map[]}] map[] map[] [call_c0e9aa514886493daba58275] map[call_c0e9aa514886493daba58275:tool 'operator_query' interrupted with arguments '{"name":"维什戴尔"}', waiting for your approval, please answer with Y/N]}, IsRootCause=false, State=interrupt state: State=&{assistant: 
reasoning content:
好的，我已经加载了operator-info技能。现在我应该查询干员维什戴尔的信息。

根据技能说明，我可以使用operator_query工具来查询干员的基本信息，或者如果需要更详细的信息，可以调用联网搜索工具。

让我先使用operator_query工具来获取维什戴尔的基本信息。
tool_calls:
index[0]:{Index:0xc000298430 ID:call_c0e9aa514886493daba58275 Type:function Function:{Name:operator_query Arguments:{"name":"维什戴尔"}} Extra:map[]}

finish_reason: tool_calls
usage: &{9049 {0} 79 9128 {64}} map[] map[] [call_c0e9aa514886493daba58275]}, LayerSpecificPayload=<nil>, SubsLen=1
2026/04/18 21:50:59 [trace] Graph/ReAct error: interrupt happened, info: &{State:0xc0004868c0 BeforeNodes:[] AfterNodes:[] RerunNodes:[ToolNode] RerunNodesExtra:map[ToolNode:0xc000c86000] SubGraphs:map[] InterruptContexts:[]}
2026/04/18 21:50:59 [trace] Chain/Amiya error: interrupt happened, info: &{State:<nil> BeforeNodes:[] AfterNodes:[] RerunNodes:[] RerunNodesExtra:map[] SubGraphs:map[node_1:0xc00076c180] InterruptContexts:[0xc000980a50]}

⚠️  需要用户审批 ⚠️
工具: operator_query
参数: {"name":"维什戴尔"}

是否批准? (y/n): y
✓ 已批准，正在执行...
2026/04/18 21:51:01 [trace] Agent/Amiya start
2026/04/18 21:51:01 [trace] Agent/Amiya end
2026/04/18 21:51:01 [trace] ToolsNode/ToolNode start
2026/04/18 21:51:01 [trace] Tool/operator_query start
2026/04/18 21:51:01 [trace] Tool/operator_query end
[调用工具结果] 未找到名为'维什戴尔'的干员记录
2026/04/18 21:51:01 [trace] ChatModel/ChatModel start
博士，本地数据库中没有找到维什戴尔的记录，让我联网搜索一下这位干员的详细信息。
[调用工具] web_search({"query":"明日方舟 维什戴尔 干员信息"})
2026/04/18 21:51:02 [trace] ToolsNode/ToolNode start

2026/04/18 21:51:02 [trace] ToolsNode/ToolNode error: interrupt signal: ID=c11c00e5-eafe-4679-8829-033562939b84, Addr=agent:Amiya;runnable:Amiya;node:node_1;node:ToolNode, Info=interrupt info: Info=&{[{0xc000298558 call_ad04446a08b44c729c617362 function {web_search {"query":"明日方舟 维什戴尔 干员信息"}} map[]}] map[] map[] [call_ad04446a08b44c729c617362] map[call_ad04446a08b44c729c617362:tool 'web_search' interrupted with arguments '{"query":"明日方舟 维什戴尔 干员信息"}', waiting for your approval, please answer with Y/N]}, IsRootCause=false, State=interrupt state: State=&{assistant: 博士，本地数据库中没有找到维什戴尔的记录，让我联网搜索一下这位干员的详细信息。
reasoning content:
看起来operator_query工具中没有找到维什戴尔的记录。维什戴尔应该是一个干员，可能是新干员或者数据库中还没有这个干员。

根据技能说明："当你缺少某些信息时，可以调用联网搜索工具"。

让我使用联网搜索来查询维什戴尔的信息。
tool_calls:
index[0]:{Index:0xc000298558 ID:call_ad04446a08b44c729c617362 Type:function Function:{Name:web_search Arguments:{"query":"明日方舟 维什戴尔 干员信息"}} Extra:map[]}

finish_reason: tool_calls
usage: &{9145 {8591} 104 9249 {62}} map[] map[] [call_ad04446a08b44c729c617362]}, LayerSpecificPayload=<nil>, SubsLen=1
2026/04/18 21:51:02 [trace] Graph/ReAct error: interrupt happened, info: &{State:0xc000345880 BeforeNodes:[] AfterNodes:[] RerunNodes:[ToolNode] RerunNodesExtra:map[ToolNode:0xc000980ff0] SubGraphs:map[] InterruptContexts:[]}
2026/04/18 21:51:02 [trace] Chain/Amiya error: interrupt happened, info: &{State:<nil> BeforeNodes:[] AfterNodes:[] RerunNodes:[] RerunNodesExtra:map[] SubGraphs:map[node_1:0xc00076d180] InterruptContexts:[0xc0009816d0]}

⚠️  需要用户审批 ⚠️
工具: web_search
参数: {"query":"明日方舟 维什戴尔 干员信息"}

是否批准? (y/n): y
✓ 已批准，正在执行...
2026/04/18 21:51:04 [trace] Agent/Amiya start
2026/04/18 21:51:04 [trace] Agent/Amiya end
2026/04/18 21:51:04 [trace] ToolsNode/ToolNode start
2026/04/18 21:51:04 [trace] Tool/web_search start
2026/04/18 21:51:04 [trace] Workflow/ start
2026/04/18 21:51:04 [trace] Lambda/ start
2026/04/18 21:51:04 [web_search] 开始搜索: 明日方舟 维什戴尔 干员信息
2026/04/18 21:51:05 [web_search] DuckDuckGo 响应状态码: 200
2026/04/18 21:51:05 [web_search] DuckDuckGo 响应长度: 34170 字节
2026/04/18 21:51:05 [web_search] 正则匹配到 10 个结果链接
2026/04/18 21:51:05 [web_search] 解析结果: 维什戴尔 - PRTS - 玩家共同构筑的明日方舟中文Wiki -> https://prts.wiki/w/%E7%BB%B4%E4%BB%80%E6%88%B4%E5%B0%94
2026/04/18 21:51:05 [web_search] 解析结果: 维什戴尔 - 百度百科 -> https://baike.baidu.com/item/%E7%BB%B4%E4%BB%80%E6%88%B4%E5%B0%94/64343969
2026/04/18 21:51:05 [web_search] 解析结果: 维什戴尔-快懂百科 - m.baike.com -> https://m.baike.com/wikiid/7369027505632870463
2026/04/18 21:51:05 [web_search] 解析结果: 维什戴尔 - 明日方舟wiki_bwiki_哔哩哔哩 -> https://wiki.biligame.com/arknights/%E7%BB%B4%E4%BB%80%E6%88%B4%E5%B0%94
2026/04/18 21:51:05 [web_search] 解析结果: 【明日方舟】干员介绍-维什戴尔_游戏热门视频 -> https://www.bilibili.com/video/BV1VrqAYAELn/
2026/04/18 21:51:05 [web_search] 解析结果: W - 萌娘百科 万物皆可萌的百科全书 -> https://mzh.moegirl.org.cn/W(%E6%98%8E%E6%97%A5%E6%96%B9%E8%88%9F)
2026/04/18 21:51:05 [web_search] 解析结果: 维什戴尔 - 明日方舟 -> https://ak.hypergryph.com/archive/dynamicCompile/202404049.html
2026/04/18 21:51:05 [web_search] 解析结果: 明日方舟维什戴尔干员图鉴 六星投掷手介绍_游戏狗手机版 -> https://m.gamedog.cn/games/a/3377674.html
2026/04/18 21:51:05 [web_search] DuckDuckGo 解析出 8 条结果
2026/04/18 21:51:05 [web_search] DuckDuckGo 返回 8 条结果
2026/04/18 21:51:08 [trace] Workflow/ end
2026/04/18 21:51:09 [trace] Lambda/ end
2026/04/18 21:51:09 [trace] Workflow/ end
2026/04/18 21:51:09 [trace] Batch/ end
2026/04/18 21:51:09 [trace] Lambda/ end
2026/04/18 21:51:09 [trace] Lambda/ start
2026/04/18 21:51:09 [trace] ChatModel/ start
    (...)
    (... 该trace日志数量过多，此处省略)
    (...)
2026/04/18 21:59:05 [trace] Workflow/ end
2026/04/18 21:59:05 [trace] Tool/web_search end
[调用工具结果] {"answer":"根据搜索结果，明日方舟干员“维什戴尔”的详细信息如下：\n\n**1. 基本信息与定位**\n*   **身份**：维什戴尔原名“W”，是六星狙击干员，分��...
2026/04/18 21:59:05 [trace] ChatModel/ChatModel start
博士，为您查询到干员维什戴尔的详细信息：

**基本信息**
- 本名：W
- 星级：★★★★★★（6星）
- 职业：狙击 - 投掷手分支
- 编号：B00W
- 种族：萨卡兹
- 出身：卡兹戴尔
- 身高：165cm

**外貌特征**
- 红瞳，银发配黑色发梢
- 红色尖角
- 身穿红色外套和黑色短裤
- 拥有末端带刺的黑色尾巴

**感染状况**
- 为矿石病感染者
- 体细胞与源石融合率：17%
- 血液源石结晶密度：0.32u/L

**战斗特性**
- 攻击对小范围地面敌人造成两次物理伤害
- 技能机制包含弹药类技能

**主要技能**
1. **一技能"定点清算"**：造成晕眩及余震
2. **二技能"饱和复仇"**：提升攻速与攻击力
3. **三技能"爆裂黎明"**：召唤魂灵之影，获得迷彩，扩大溅射范围并实现空攻击

**社区风评**
维什戴尔被社区誉为**唯一真神**和**强度的顶点**，是非常强大的六星狙击干员。

**背景故事**
- 曾担任萨卡兹雇佣兵领袖及巴别塔成员
- 在特雷西娅遇刺后，她追杀刺客
- 最终更名维什戴尔并加入罗德岛成为精英干员

维什戴尔是一名限定干员，拥有专属模组，是非常强大的六星狙击干员。博士，如果需要了解更多关于她的具体使用技巧或材料需求，请告诉我哦。
```