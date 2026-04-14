package agent

import (
	"AmiyaAgent/internal/logic"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

func InitTools() []tool.BaseTool {
	var tools []tool.BaseTool

	OperatorQueryTool, err := utils.InferTool(
		"operator_query",
		"根据干员名称查询其职业、特长和天赋等基本信息",
		logic.QueryOperator,
	)
	if err != nil {
		log.Print("初始化工具失败:", err)
	}
	log.Println("工具 operator_query 初始化成功")
	tools = append(tools, OperatorQueryTool)

	ResourceQueryTool, err := utils.InferTool(
		"resource_query",
		"查询当前的资源状况，包括龙门币、合成玉、理智、源石等",
		logic.QueryResources,
	)
	if err != nil {
		log.Print("初始化工具失败:", err)
	}
	log.Println("工具 resource_query 初始化成功")
	tools = append(tools, ResourceQueryTool)

	BattlePlanTool, err := utils.InferTool(
		"battle_plan",
		"根据关卡名称和难度偏好制定作战计划，推荐编队和作战要点",
		logic.MakeBattlePlan,
	)
	if err != nil {
		log.Print("初始化工具失败:", err)
	}
	log.Println("工具 battle_plan 初始化成功")
	tools = append(tools, BattlePlanTool)

	return tools
}
