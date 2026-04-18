> 该README由AI生成，大佬勿嘲笑qwq

# AmiyaAgent

基于 [CloudWeGo Eino](https://github.com/cloudwego/eino) 框架构建的明日方舟智能助手 Agent，以罗德岛领袖阿米娅的形象为博士提供游戏辅助服务。

## 功能特性

### 核心能力

- **角色扮演**: 以阿米娅的身份与博士进行自然对话，温柔认真地提供帮助
- **工具调用**: 支持多种内置工具，可查询干员信息、资源状况、制定作战计划
- **联网搜索**: 支持实时联网搜索，获取最新的明日方舟资讯和攻略
- **文档问答**: RAG 能力支持，可基于上传文档进行智能问答
- **技能系统**: 可扩展的技能框架，支持自定义领域知识

### 内置工具

| 工具名称 | 功能描述 |
|---------|---------|
| `operator_query` | 根据干员名称查询其职业、特长和天赋等基本信息 |
| `resource_query` | 查询当前的资源状况，包括龙门币、合成玉、理智、源石等 |
| `battle_plan` | 根据关卡名称和难度偏好制定作战计划，推荐编队和作战要点 |
| `answer_from_document` | 在上传文档中搜索相关内容并合成答案（RAG） |
| `web_search` | 联网搜索工具，获取实时信息和最新数据 |

### 技能系统

项目内置了 4 个明日方舟相关技能：

| 技能名称 | 功能描述 |
|---------|---------|
| `operator-info` | 干员信息查询 - 查询干员的详细信息，包括职业、星级、技能、天赋、基建技能等 |
| `stage-guide` | 关卡攻略 - 提供主线、活动、危机合约等关卡的打法建议 |
| `team-builder` | 编队推荐 - 根据关卡需求推荐干员组合和阵容搭配 |
| `material-planner` | 材料规划 - 计算干员培养所需材料，提供资源规划建议 |

## 项目结构

```
AmiyaAgent/
├── cmd/
│   └── main.go              # 应用入口
├── config/
│   └── config.yaml          # 配置文件
├── docs/
│   └── TERRA A JOURNEY.txt  # RAG示例文档
├── internal/
│   ├── agent/
│   │   ├── agent.go         # Agent 核心逻辑
│   │   ├── prompt.go        # 角色设定和提示词
│   │   └── tools.go         # 工具初始化
│   ├── component/
│   │   ├── memory.go        # 会话存储
│   │   ├── middleware.go    # 中间件（审批、安全）
│   │   └── model.go         # ChatModel 初始化
│   ├── graphTool/
│   │   ├── rag.go           # RAG 文档问答工具
│   │   └── websearch.go     # 联网搜索工具
│   ├── logic/
│   │   └── tools.go         # 业务逻辑工具实现
│   └── repository/
│       └── dailyLuck.go     # 数据存储
├── skills/
│   ├── operator-info/       # 干员信息技能
│   ├── stage-guide/         # 关卡攻略技能
│   ├── team-builder/        # 编队推荐技能
│   └── material-planner/    # 材料规划技能
├── sessionStore/            # 会话存储目录
├── go.mod
├── go.sum
└── README.md
```

## 运行示例

```
PS D:\CodeStudy\GoProjects\src\AmiyaAgent> go run ./cmd/main.go
2026/04/18 21:50:32 ChatModel 实例创建成功
2026/04/18 21:50:32 CozeLoop追踪已启用
2026/04/18 21:50:32 工具 operator_query 初始化成功
2026/04/18 21:50:32 工具 resource_query 初始化成功
2026/04/18 21:50:32 工具 battle_plan 初始化成功
2026/04/18 21:50:32 工具 rag 初始化成功
2026/04/18 21:50:32 工具 web_search 初始化成功
2026/04/18 21:50:37 DeepAgent 实例创建成功
2026/04/18 21:50:37 AgentRunner 实例创建成功

===============================================开始对话（当前会话标题：新会话）=======================================================

阿米娅: 博士，您好！我是阿米娅，有什么需要我帮忙的吗？

博士：查询干员维什戴尔的信息
阿米娅：博士，为您查询到干员维什戴尔的详细信息：

**基本信息**
- 本名：W
- 星级：★★★★★★（6星）
- 职业：狙击 - 投掷手分支
- 编号：B00W
- 种族：萨卡兹
- 出身：卡兹戴尔

**战斗特性**
- 攻击对小范围地面敌人造成两次物理伤害
- 技能机制包含弹药类技能

**社区风评**
维什戴尔被社区誉为**唯一真神**和**强度的顶点**，是非常强大的六星狙击干员。
```

## 依赖

- [github.com/cloudwego/eino](https://github.com/cloudwego/eino) - AI Agent 框架
- [github.com/cloudwego/eino-ext](https://github.com/cloudwego/eino-ext) - Eino 扩展组件
- [github.com/coze-dev/cozeloop-go](https://github.com/coze-dev/cozeloop-go) - CozeLoop 追踪
- [github.com/joho/godotenv](https://github.com/joho/godotenv) - 环境变量管理
