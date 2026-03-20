package tool

import (
	"context"
	"encoding/json"
)

// ThinkTool provides a dedicated space for complex reasoning.
// Reference: Anthropic Engineering - "The 'think' tool" article
//
// When to use:
// - Analyzing multiple leads and prioritizing
// - Formulating job application strategy
// - Evaluating offers
// - Making decisions that require gathering information first
//
// Why it works:
// - Gives the model "breathing room" to process tool outputs
// - Particularly effective for sequential decision-making where each step builds on previous ones
// - In τ-bench tests, improved pass^1 from 0.370 to 0.570 (54% relative improvement)
type ThinkTool struct{}

// ThinkToolInput represents the input for the think tool.
type ThinkToolInput struct {
	Thought string `json:"thought"`
}

func (t *ThinkTool) Definition() Definition {
	return Definition{
		Name: "think",
		Description: `用于复杂推理的思考工具。

适用场景：
- 分析多个线索的优先级，制定投递策略
- 评估收到的 Offer，对比各机会
- 整理从工具获取的信息，做出决策
- 处理需要多步推理的任务

使用方法：
直接输出你的思考内容，这个工具不会返回新信息，只是让你停下来整理思路。

注意：如果你已经有足够的信息直接回答用户，就不需要调用这个工具。`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"thought": map[string]any{
					"type":        "string",
					"description": "你的思考内容。应该包含：你收集到了什么信息、接下来要做什么、你的推理过程",
				},
			},
			"required": []string{"thought"},
		},
	}
}

func (t *ThinkTool) Run(_ context.Context, input json.RawMessage) (string, error) {

	var args ThinkToolInput
	if err := json.Unmarshal(input, &args); err != nil {
		return "", err
	}

	// The think tool simply acknowledges the thought and echoes it back
	// This serves as a "scratchpad" - the model can review its own thinking
	// Reference: Anthropic's implementation just logs the thought
	result := map[string]any{
		"acknowledged": true,
		"thought":      args.Thought,
		"note":         "已记录你的思考。继续基于这个思路行动吧。",
	}

	encoded, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
