package logic

import (
	"context"
	"fmt"
	"strings"
)

type OperatorQueryToolParams struct {
	Name string `json:"name" jsonschema:"description=干员名称，如'能天使'、'银灰'等"`
}

// 查询干员信息伪逻辑
func QueryOperator(ctx context.Context, input *OperatorQueryToolParams) (string, error) {
	operators := map[string]string{
		"能天使":  "【能天使】★★★★★★ | 职业：狙击 | 特长：对空攻击，高射速 | 天赋：攻击速度提升",
		"银灰":   "【银灰】★★★★★★ | 职业：近卫 | 特长：远程斩击 | 天赋：再部署时间缩短",
		"陈":    "【陈】★★★★★★ | 职业：近卫 | 特长：多段爆发攻击 | 天赋：技力恢复速度提升",
		"艾雅法拉": "【艾雅法拉】★★★★★★ | 职业：术师 | 特长：群体法术伤害 | 天赋：法术伤害提升",
		"阿米娅":  "【阿米娅】★★★★★★ | 职业：术师/近卫 | 特长：真实伤害 | 天赋：每次攻击回复技力",
		"德克萨斯": "【德克萨斯】★★★★★ | 职业：先锋 | 特长：产生费用，群体眩晕 | 天赋：初始Cost+1",
		"蓝毒":   "【蓝毒】★★★★★ | 职业：狙击 | 特长：多目标攻击，法术dot伤害 | 天赋：攻击附带法术伤害",
	}

	info, ofk := operators[input.Name]
	if !ofk {
		return fmt.Sprintf("未找到名为'%s'的干员记录", input.Name), nil
	}

	return info, nil
}

type ResourceQueryToolParams struct {
	ResourceType string `json:"resource_type" jsonschema:"description=资源类型，可选值：龙门币、合成玉、理智、源石、全部"`
}

// 查询资源状况伪逻辑
func QueryResources(ctx context.Context, input *ResourceQueryToolParams) (string, error) {
	resources := map[string]string{
		"龙门币": "龙门币: 1,234,567",
		"合成玉": "合成玉: 8,900",
		"理智":  "理智: 130/135（将在 2 小时后回满）",
		"源石":  "源石: 42 个",
	}

	if input.ResourceType == "全部" || input.ResourceType == "" {
		var result strings.Builder
		result.WriteString("=== 罗德岛资源报告 ===\n")
		for _, v := range resources {
			result.WriteString("  " + v + "\n")
		}
		result.WriteString("报告时间：刚刚更新")
		return result.String(), nil
	}

	info, ok := resources[input.ResourceType]
	if !ok {
		return fmt.Sprintf("未知的资源类型: '%s'。支持查询：龙门币、合成玉、理智、源石、全部", input.ResourceType), nil
	}

	return info, nil
}

type BattlePlanToolParams struct {
	StageName  string `json:"stage_name" jsonschema:"description=关卡名称，如'1-7'、'CE-5'、'SK-5'等"`
	Difficulty string `json:"difficulty" jsonschema:"description=难度偏好，可选值：标准、挑战"`
}

// 制定作战计划伪逻辑
func MakeBattlePlan(ctx context.Context, input *BattlePlanToolParams) (string, error) {
	plan := fmt.Sprintf(`=== 作战计划：%s（%s模式）===
推荐编队：
  先锋位：德克萨斯 — 快速获取部署费用
  狙击位：能天使 — 对空防御 + 高DPS
  近卫位：银灰 — 核心输出，关键时刻真银斩
  术师位：艾雅法拉 — 群体法术清场
  医疗位：闪灵 — 全队治疗保障
  重装位：星熊 — 前排抗压

作战要点：
  1. 开局立即部署德克萨斯获取初始费用
  2. 优先在高台部署能天使控制空中威胁
  3. 银灰部署在关键路口，技能 3 留到敌方精英出现时使用
  4. 艾雅法拉在敌方密集时释放火山，最大化群伤

预估理智消耗：%d
预估通关概率：%s`, input.StageName, input.Difficulty, 30, "95%")

	return plan, nil
}
