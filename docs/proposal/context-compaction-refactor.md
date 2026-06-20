# 上下文压缩逻辑重构方案

## 1. 当前实现分析

### 1.1 核心组件

**文件结构**:
- `internal/context/compaction.go` - 压缩逻辑
- `internal/context/context.go` - Token估算和上下文管理
- `internal/agent/agent.go` - Agent层压缩调用
- `internal/session/session.go` - Session层压缩记录

**主要功能**:
1. Token估算 (`EstimateTokens`)
2. 压缩触发判断 (`ShouldCompact`)
3. 切割点查找 (`FindCutPoint`)
4. 摘要生成 (`GenerateSummaryInsertThenCompress`)
5. 压缩执行 (`Compact`)

### 1.2 当前流程

```
1. 每次LLM调用后检查 ShouldCompact()
2. 如果触发，调用 Compact()
3. FindCutPoint() 找到切割点（保留最近的KeepRecentTokens）
4. 将切割点之前的旧messages作为压缩输入
5. 使用Insert-then-Compress模式生成或更新摘要
6. 用摘要替换旧messages，保留切割点之后的最近messages
7. 保存压缩记录到Session
```

---

## 2. 问题识别

### 2.1 Token估算不准确

**问题**:
- 使用 `chars/4` 启发式估算，误差较大
- 不同模型的tokenizer不同，统一估算不准确
- 图片使用固定4800字符估算，不考虑实际分辨率

**影响**:
- 压缩时机不准确
- 可能过早或过晚触发压缩
- 浪费context window或频繁压缩

### 2.2 压缩成本仍可优化

**问题**:
- 每次压缩需要发送切割点之前的旧消息段给LLM
- 压缩本身消耗大量tokens
- 增量边界和摘要覆盖范围没有显式元数据

**影响**:
- 压缩延迟高
- API调用成本高
- 用户体验差（压缩时等待时间长）

### 2.3 压缩质量不可控

**问题**:
- 压缩指令硬编码，无法定制
- 没有压缩质量评估
- 摘要格式固定，可能不适合所有场景

**影响**:
- 重要信息可能丢失
- 摘要可能过于冗长或简短
- 无法根据任务类型调整压缩策略

### 2.4 缺少增量压缩

**问题**:
- 当前会使用 `previousSummary` 更新摘要，但没有显式记录摘要覆盖到哪个entry
- 多轮压缩依赖消息序列中的summary识别，缺少稳定的compaction metadata
- Session层只保存最终summary和first kept entry，无法清晰表达摘要版本关系

**影响**:
- 多轮压缩行为不够透明
- 后续排查和回放压缩状态困难
- 增量优化缺少可靠边界

### 2.5 切割策略简单

**问题**:
- 主要基于token数量切割
- 已有turn边界保护，但对tool call/result配对和重要信息缺少更明确的保护策略
- 切割决策缺少可观测信息，难以判断是否丢失关键上下文

**影响**:
- 上下文连贯性受损
- 重要信息可能被切割
- 摘要质量下降

### 2.6 Idle压缩残留代码

**问题**:
- 配置项已定义（`IdleCompressionEnabled`, `IdleTimeoutSeconds`, `IdleMinTokensForCompress`）
- 但没有实际实现逻辑
- 代码和文档中存在无效配置

**影响**:
- 配置混乱
- 维护负担

---

## 3. 重构方案

### 3.1 Phase 1: Token估算改进

#### 3.1.1 集成Tokenizer

**目标**: 使用准确的tokenizer替代chars/4启发式

**实现**:
```go
// internal/context/tokenizer.go
type Tokenizer interface {
    EstimateTokens(text string) int
    EstimateMessagesTokens(messages []provider.Message) int
}

// 针对不同模型的tokenizer实现
type OpenAITokenizer struct{ /* tiktoken */ }
type AnthropicTokenizer struct{ /* claude tokenizer */ }
type GenericTokenizer struct{ /* chars/4 fallback */ }
```

**配置**:
```json
{
  "compaction": {
    "tokenizer": "auto",  // auto, openai, anthropic, generic
    "tokenizerModel": "claude-sonnet-4-20250514"
  }
}
```

**优先级**: 高 - 影响压缩准确性

#### 3.1.2 图片Token优化

**目标**: 根据图片分辨率和模型估算图片tokens

**实现**:
```go
func EstimateImageTokens(image *provider.ImageContent, model string) int {
    // 根据模型和图片尺寸计算
    // OpenAI: 基于tile数量
    // Anthropic: 基于分辨率
    // Fallback: 固定估算
}
```

**优先级**: 中

### 3.2 Phase 2: 增量压缩

#### 3.2.1 增量摘要更新

**目标**: 只压缩新增内容，更新现有摘要

**实现**:
```go
func CompactIncremental(
    ctx context.Context,
    messages []provider.Message,
    previousSummary string,
    newMessages []provider.Message,  // 只包含新增的messages
    p provider.Provider,
    model *provider.Model,
    systemPrompt string,
    tools []provider.ToolDefinition,
    settings CompactionSettings,
) (*CompactionResult, error) {
    // 1. 只序列化newMessages
    // 2. 使用updateCompressionInstruction
    // 3. 生成增量摘要
    // 4. 合并到previousSummary
}
```

**优势**:
- 减少处理的messages数量
- 降低API调用成本
- 提高压缩速度

**优先级**: 高 - 显著降低成本

#### 3.2.2 摘要版本管理

**目标**: 跟踪摘要的演变，支持回滚

**实现**:
```go
type SummaryVersion struct {
    Version   int
    Summary   string
    Timestamp time.Time
    MessageIDs []string  // 被压缩的message IDs
}

// Session中保存摘要历史
type CompactionEntry struct {
    EntryBase
    Summary        string
    FirstKeptEntry string
    TokensBefore   int
    SummaryVersion int    // 新增
    ParentSummary  string // 新增：之前的摘要（用于回滚）
}
```

**优先级**: 中

### 3.3 Phase 3: 智能切割策略

#### 3.3.1 语义边界检测

**目标**: 在语义边界切割，保持上下文连贯

**实现**:
```go
type CutStrategy int

const (
    CutByTokens CutStrategy = iota  // 当前策略
    CutByTurn                       // 在turn边界切割
    CutByTopic                      // 在主题边界切割（需要LLM判断）
    CutHybrid                       // 混合策略
)

func FindCutPointSemantic(
    messages []provider.Message,
    strategy CutStrategy,
    keepRecentTokens int,
    p provider.Provider,  // 用于topic检测
) CutPointResult {
    switch strategy {
    case CutByTokens:
        return FindCutPointByTokens(messages, keepRecentTokens)
    case CutByTurn:
        return FindCutPointByTurn(messages, keepRecentTokens)
    case CutByTopic:
        return FindCutPointByTopic(messages, keepRecentTokens, p)
    case CutHybrid:
        return FindCutPointHybrid(messages, keepRecentTokens, p)
    }
}
```

**配置**:
```json
{
  "compaction": {
    "cutStrategy": "hybrid",  // tokens, turn, topic, hybrid
    "preferTurnBoundary": true,
    "topicDetectionThreshold": 0.7
  }
}
```

**优先级**: 中 - 提高压缩质量

#### 3.3.2 重要信息保护

**目标**: 保护关键信息不被切割

**实现**:
```go
type MessageImportance int

const (
    ImportanceLow MessageImportance = iota
    ImportanceMedium
    ImportanceHigh
    ImportanceCritical
)

// 标记重要messages
func MarkImportance(messages []provider.Message) []MessageWithImportance {
    // 1. 包含错误信息的message -> ImportanceHigh
    // 2. 包含文件路径的message -> ImportanceMedium
    // 3. 包含代码片段的message -> ImportanceMedium
    // 4. 用户明确标记的message -> ImportanceCritical
}

// 切割时保护重要messages
func FindCutPointWithProtection(
    messages []provider.Message,
    keepRecentTokens int,
    protectThreshold MessageImportance,
) CutPointResult {
    // 跳过importance >= protectThreshold的messages
}
```

**配置**:
```json
{
  "compaction": {
    "protectImportance": "high",  // low, medium, high, critical
    "protectPatterns": ["error", "file:", "function "]
  }
}
```

**优先级**: 中

### 3.4 Phase 4: 压缩质量优化

#### 3.4.1 可配置压缩指令

**目标**: 允许用户自定义压缩指令

**实现**:
```go
// internal/context/templates.go
type CompressionTemplate struct {
    Name        string
    Instruction string
    UpdateInstruction string
}

// 内置模板
var builtinTemplates = map[string]CompressionTemplate{
    "default": {
        Name: "default",
        Instruction: `Please create a structured context checkpoint summary...`,
        UpdateInstruction: `Please update the existing summary...`,
    },
    "code": {
        Name: "code",
        Instruction: `Focus on code changes, file paths, and technical decisions...`,
        UpdateInstruction: `Update the summary focusing on recent code changes...`,
    },
    "conversation": {
        Name: "conversation",
        Instruction: `Focus on conversation flow, decisions made, and action items...`,
        UpdateInstruction: `Update the summary with new conversation points...`,
    },
}

// 用户自定义模板（从配置加载）
func LoadCustomTemplates(path string) map[string]CompressionTemplate {
    // 从JSON/YAML文件加载
}
```

**配置**:
```json
{
  "compaction": {
    "template": "default",  // 或 "code", "conversation", 自定义名称
    "customTemplatePath": ".vibe/compression-templates.json"
  }
}
```

**优先级**: 中

#### 3.4.2 压缩质量评估

**目标**: 评估压缩质量，支持重试

**实现**:
```go
type CompactionQuality struct {
    Completeness float64  // 0-1，信息完整度
    Conciseness  float64  // 0-1，简洁度
    Relevance    float64  // 0-1，相关性
    Overall      float64  // 综合评分
}

func EvaluateCompactionQuality(
    original []provider.Message,
    summary string,
    p provider.Provider,
    model *provider.Model,
) (*CompactionQuality, error) {
    // 使用LLM评估压缩质量
    // 返回评分和改进建议
}

func CompactWithQualityCheck(
    ctx context.Context,
    messages []provider.Message,
    minQuality float64,  // 最低质量要求
    maxRetries int,
    p provider.Provider,
    model *provider.Model,
    systemPrompt string,
    tools []provider.ToolDefinition,
    settings CompactionSettings,
    previousSummary string,
) (*CompactionResult, error) {
    for i := 0; i < maxRetries; i++ {
        result, err := Compact(ctx, messages, p, model, systemPrompt, tools, settings, previousSummary)
        if err != nil {
            continue
        }
        
        quality, err := EvaluateCompactionQuality(messages[:result.FirstKeptIndex], result.Summary, p, model)
        if err != nil {
            return result, nil  // 评估失败，接受结果
        }
        
        if quality.Overall >= minQuality {
            return result, nil
        }
        
        // 质量不达标，重试
    }
    
    return nil, fmt.Errorf("failed to achieve minimum quality after %d retries", maxRetries)
}
```

**配置**:
```json
{
  "compaction": {
    "qualityCheck": {
      "enabled": false,
      "minQuality": 0.7,
      "maxRetries": 2
    }
  }
}
```

**优先级**: 低 - 可选功能

### 3.5 配套清理：Idle压缩残留

#### 3.5.1 处理Idle压缩残留

**目标**: 消除未实现配置带来的误导

**任务**:
1. 明确选择实现最小 idle 压缩，或将 idle 配置标记为 deprecated
2. 如果选择 deprecated，保留字段解析，避免破坏已有 `settings.json`
3. 更新测试文件
4. 更新文档，说明当前行为和迁移路径

**优先级**: 高 - 减少配置混乱

---

## 4. 实施计划

### 4.1 Phase 1: Token估算改进（1-2周）

**任务**:
1. 实现Tokenizer接口
2. 集成tiktoken（OpenAI）和claude-tokenizer（Anthropic）
3. 实现图片token估算优化
4. 添加配置选项
5. 更新测试

**验收标准**:
- Token估算误差 < 10%
- 支持主流模型
- 向后兼容

### 4.2 Phase 2: 增量压缩（1-2周）

**任务**:
1. 实现增量摘要更新逻辑
2. 优化`GenerateSummaryInsertThenCompress`支持增量
3. 添加摘要版本管理
4. 更新Session存储
5. 测试增量压缩效果

**验收标准**:
- 增量压缩比全量压缩快50%+
- 摘要质量不下降
- 支持回滚到历史版本

### 4.3 Phase 3: 智能切割策略（2-3周）

**任务**:
1. 实现多种切割策略
2. 语义边界检测（基于turn）
3. 重要信息保护机制
4. 配置选项
5. 测试和调优

**验收标准**:
- 语义边界检测准确率 > 80%
- 重要信息保护有效
- 性能影响可接受

### 4.4 Phase 4: 压缩质量优化（2-3周）

**任务**:
1. 实现可配置压缩指令
2. 实现压缩质量评估
3. 支持重试机制
4. 用户自定义模板
5. 文档和示例

**验收标准**:
- 压缩质量可配置
- 质量评估准确
- 用户满意度提升

### 4.5 配套清理：Idle压缩残留（1周）

**任务**:
1. 处理Idle压缩残留配置
2. 清理无效代码或标记弃用路径
3. 更新文档
4. 测试验证

**验收标准**:
- 配置行为明确
- 文档准确
- 无功能影响

---

## 5. 配置设计

### 5.1 完整配置示例

```json
{
  "compaction": {
    "enabled": true,
    "reserveTokens": 16384,
    "keepRecentTokens": 20000,
    
    // Token估算
    "tokenizer": "auto",
    "tokenizerModel": "claude-sonnet-4-20250514",
    
    // 切割策略
    "cutStrategy": "hybrid",
    "preferTurnBoundary": true,
    "protectImportance": "high",
    "protectPatterns": ["error", "file:", "function "],
    
    // 压缩质量
    "template": "default",
    "customTemplatePath": ".vibe/compression-templates.json",
    "qualityCheck": {
      "enabled": false,
      "minQuality": 0.7,
      "maxRetries": 2
    }
  }
}
```

### 5.2 配置验证

```go
func ValidateCompactionSettings(settings CompactionSettings) error {
    if settings.ReserveTokens < 1024 {
        return fmt.Errorf("reserveTokens must be >= 1024")
    }
    if settings.KeepRecentTokens < 1000 {
        return fmt.Errorf("keepRecentTokens must be >= 1000")
    }
    // ... 更多验证
    return nil
}
```

---

## 6. 测试策略

### 6.1 单元测试

- Token估算准确性测试
- 切割策略测试
- 增量压缩测试
- 配置验证测试

### 6.2 集成测试

- 端到端压缩流程
- Session持久化
- 多轮压缩
- 错误恢复

### 6.3 性能测试

- 压缩延迟基准
- Token估算性能
- 内存占用

### 6.4 质量测试

- 摘要质量评估
- 信息保留率
- 用户满意度调查

---

## 7. 风险和缓解

### 7.1 技术风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Tokenizer集成复杂 | 延期 | 先实现fallback，逐步集成 |
| 增量压缩质量下降 | 用户体验差 | 质量评估+重试机制 |
| 语义边界检测不准 | 信息丢失 | 提供多种策略选择 |

### 7.2 兼容性风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 配置格式变化 | 用户配置失效 | 向后兼容，自动迁移 |
| Session格式变化 | 历史数据不可用 | 版本管理，迁移工具 |
| API接口变化 | 集成方受影响 | 保持接口稳定，渐进式变更 |

### 7.3 资源风险

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 开发周期长 | 延期 | 分阶段交付，MVP优先 |
| 测试覆盖不足 | 质量问题 | 自动化测试，持续集成 |
| 文档滞后 | 用户困惑 | 文档先行，及时更新 |

---

## 8. 成功指标

### 8.1 技术指标

- Token估算误差: < 10%
- 压缩延迟: 降低50%+
- 增量压缩速度: 提升70%+

### 8.2 用户体验指标

- 压缩等待时间: < 5秒（增量）, < 15秒（全量）
- 信息保留率: > 90%
- 用户满意度: > 4.0/5.0
- 压缩相关问题反馈: 减少50%+

### 8.3 业务指标

- API调用成本: 降低30%+
- 开发效率: 提升20%+
- 文档完整性: 100%
- 测试覆盖率: > 80%

---

## 9. 总结

本重构方案聚焦4个核心阶段逐步优化上下文压缩逻辑：Token估算、增量压缩、智能切割和质量优化；Idle配置作为配套清理处理。核心目标是先提升压缩触发准确性、摘要连续性、切割安全性和配置可信度，暂不引入并行压缩或压缩缓存等低优先级性能机制。

**关键改进点**:
1. ✅ 准确的Token估算
2. ✅ 增量压缩降低成本
3. ✅ 智能切割保护重要信息
4. ✅ 可配置压缩策略
5. ✅ 质量评估和重试机制
6. ✅ 清理无效配置

**预期收益**:
- 压缩成本降低30%+
- 压缩速度提升50%+
- 用户满意度提升
- 系统可维护性增强

建议按Phase顺序实施，每个Phase完成后进行验收和调整，确保质量和进度。
