import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export interface PrepGenerationConfig {
  topicKeys: string[];
  questionCount: number;
  includeResume: boolean;
  includeProfile: boolean;
  includeLeadDocs: boolean;
}

interface PrepConfigPanelProps {
  availableTopicKeys: string[];
  config: PrepGenerationConfig;
  onChange: (next: PrepGenerationConfig) => void;
  onGenerate: () => void;
  isGenerating?: boolean;
  disabled?: boolean;
}

export function GenerateQuestionsButton({
  onClick,
  disabled = false,
  isLoading = false,
}: {
  onClick: () => void;
  disabled?: boolean;
  isLoading?: boolean;
}) {
  return (
    <Button type="button" onClick={onClick} disabled={disabled || isLoading} className="w-full sm:w-auto">
      {isLoading ? "生成中..." : "生成题目"}
    </Button>
  );
}

export function PrepConfigPanel({
  availableTopicKeys,
  config,
  onChange,
  onGenerate,
  isGenerating = false,
  disabled = false,
}: PrepConfigPanelProps) {
  const handleTopicToggle = (topicKey: string, checked: boolean) => {
    const nextTopicKeys = checked
      ? Array.from(new Set([...config.topicKeys, topicKey]))
      : config.topicKeys.filter((item) => item !== topicKey);
    onChange({ ...config, topicKeys: nextTopicKeys });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">出题配置</CardTitle>
        <CardDescription>选择 topic 和上下文范围，然后生成一轮结构化面试题。</CardDescription>
      </CardHeader>
      <CardContent className="space-y-5">
        <div className="space-y-2">
          <Label className="text-sm">题目数量</Label>
          <Input
            type="number"
            min={1}
            max={20}
            value={config.questionCount}
            onChange={(event) => {
              const next = Number(event.target.value);
              onChange({
                ...config,
                questionCount: Number.isFinite(next) ? Math.max(1, Math.min(20, Math.floor(next))) : 1,
              });
            }}
            disabled={disabled || isGenerating}
            className="max-w-32"
          />
        </div>

        <div className="space-y-3">
          <Label className="text-sm">Topic Packs</Label>
          <div className="grid gap-2 sm:grid-cols-2">
            {availableTopicKeys.length === 0 ? (
              <p className="text-sm text-muted-foreground">暂无可选 topic，请先在资料侧维护。</p>
            ) : (
              availableTopicKeys.map((topicKey) => {
                const checked = config.topicKeys.includes(topicKey);
                return (
                  <div key={topicKey} className="flex items-center gap-2 rounded-md border border-border px-3 py-2 text-sm">
                    <Checkbox
                      checked={checked}
                      onCheckedChange={(value) => handleTopicToggle(topicKey, Boolean(value))}
                      disabled={disabled || isGenerating}
                    />
                    <span>{topicKey}</span>
                  </div>
                );
              })
            )}
          </div>
        </div>

        <div className="grid gap-2 sm:grid-cols-3">
          <div className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={config.includeResume}
              onCheckedChange={(value) => onChange({ ...config, includeResume: Boolean(value) })}
              disabled={disabled || isGenerating}
            />
            <span>包含简历</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={config.includeProfile}
              onCheckedChange={(value) => onChange({ ...config, includeProfile: Boolean(value) })}
              disabled={disabled || isGenerating}
            />
            <span>包含画像</span>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Checkbox
              checked={config.includeLeadDocs}
              onCheckedChange={(value) => onChange({ ...config, includeLeadDocs: Boolean(value) })}
              disabled={disabled || isGenerating}
            />
            <span>包含线索文档</span>
          </div>
        </div>

        <GenerateQuestionsButton
          onClick={onGenerate}
          isLoading={isGenerating}
          disabled={disabled || config.topicKeys.length === 0}
        />
      </CardContent>
    </Card>
  );
}
