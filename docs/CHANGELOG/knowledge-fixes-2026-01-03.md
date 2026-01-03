**Knowledge Generation Fixes — 2026-01-03**

- **Summary:**
  - 修复了知识库文档分块生成问题时，模型生成的问题文本丢失文档年份（例如“1993年/2018年”）的问题。
  - 清理并优化了为调试所做的临时/激进修改，恢复了更合理的注入与日志策略。

**Timeline**
- 2026-01-03: 诊断与一系列修复提交；定位核心原因并完成回归修复与代码整理。

**Root Cause**
- 在解析 LLM 返回的问句时，代码使用了 `strings.TrimLeft(line, "0123456789.-*) ")` 来去掉序号前缀。`TrimLeft` 将会删除位于开头的任意切字符集合中的字符，因此当一行以年份（如 `1993`）开始时，所有数字被误删，导致生成的题目丢失年份。

**修复与变更列表**

- 修复 1 — 精准去除列表序号
  - 文件：`internal/application/service/knowledge.go`
  - 问题（旧逻辑）：
    - `line = strings.TrimLeft(line, "0123456789.-*) ")`
  - 修复（新逻辑）：
    - 使用正则匹配只移除常见的列表标记：
      - `listMarkerRe := regexp.MustCompile(`^(\d+[\.\)]|[\-\*])\s*`)
      - `line = listMarkerRe.ReplaceAllString(line, "")`
  - 说明：仅删除像 `1.`, `1)`, `-`, `*` 等显式列表标记，保留以纯数字开头但不是序号（例如年份）的文本。

- 修复 2 — 撤回侵入式元数据注入（保持 Prompt 模板为第一责任）
  - 文件：`internal/application/service/knowledge.go`
  - 变更：移除将 `docMetadata` 直接拼接到 `content`（即删除：`content = fmt.Sprintf("【文档关键信息】：\n%s\n\n【正文内容】：\n%s", docMetadata, content)`）
  - 说明：恢复为通过模板占位符 `{{.DocMetadata}}` 注入元数据；同时保留向后兼容的回退策略（当旧模板没有 `{{.DocMetadata}}` 时，将元数据并入 `contextSection`）。这避免了把调试/强制行为硬编码到生成内容中，保持模板驱动的清晰性。

- 修复 3 — 恢复并降级日志级别
  - 文件：`internal/application/service/knowledge.go`
  - 变更：将用于打印完整 Prompt 与模型返回结果的日志从 `Infof` 降为 `Debugf`。
  - 说明：避免在正常运行时将敏感或过大的 Prompt/Response 打印到 Info 日志中，保留 Debug 级别用于排查问题。

- 其他微调
  - 保持 `extractDocumentMetadata` 的增强（用于提取 `完整标题`、`标准简称`、`日期/时间` 等），并确保其截取/上下文长度限制（例如最多 1500 字）以避免请求变得过长。
  - 保留对旧配置兼容的逻辑：当 `config` 中的 prompt 模板不含 `{{.DocMetadata}}` 时，将元数据插入 `contextSection`。

**受影响文件（已修改）**
- `internal/application/service/knowledge.go` — 主修复与清理所在文件（问句解析、元数据注入回退、日志级别调整）
- `config/config.yaml` — （未直接改动为本次修复必需，但此前有过提示词调整；当前保留模板驱动设计，建议确认模板包含 `{{.DocMetadata}}` 占位以获得最优结果）

**关键代码片段（前后对比）**
- 旧：
  - `line = strings.TrimLeft(line, "0123456789.-*) ")`
- 新：
  - `listMarkerRe := regexp.MustCompile(`^(\d+[\.\)]|[\-\*])\s*`)
  - `line = listMarkerRe.ReplaceAllString(line, "")`

- 旧（激进注入）：
  - `content = fmt.Sprintf("【文档关键信息】：\n%s\n\n【正文内容】：\n%s", docMetadata, content)`
- 新（恢复模板）：
  - 删除上行，改为：将 `docMetadata` 放入 `docMetadataSection`，通过 `{{.DocMetadata}}` 或回退插入 `contextSection`。

**验证步骤（如何在本地/测试环境确认修复）**
1. 重新构建并启动服务（在仓库根目录）：

```bash
./scripts/dev.sh start
```

2. 重新上传触发用例（例如 `Constitution Amendment (2018).md`），等待异步处理完成。
3. 查看后端日志（或直接搜索）Debug 输出，关键日志关键字：
   - `Generating questions with prompt`（此为 Debug，若需要开启 Debug 日志级别）
   - `Received response from model`（同上）
4. 在前端或导出结果查看生成的问题是否包含年份（例如：`1993年宪法修正案修改了哪些条款？`）。
5. 如果仍有差异：
   - 检查 `config` 中用于问题生成的 prompt 模板，确保包含 `{{.DocMetadata}}` 占位符，或接受回退逻辑中的上/下文显示。

**回滚与风险**
- 风险：无破坏性数据库/结构调整；主要改动在处理链路和日志级别，回滚安全。
- 回滚方法：将 `internal/application/service/knowledge.go` 恢复到之前版本（git revert 或手动恢复被删除的注入行）。

**建议与后续改进**
- 持续：保持 Prompt 模板为第一位的设计（模板应覆盖 `{{.DocMetadata}}`、`{{.Content}}`、`{{.Context}}`）。
- 增强：在 `extractDocumentMetadata` 的返回中采用结构化输出（JSON）并在本地做严格验证，减少模型自由发挥带来的不稳定性。
- 监控：在关键路径加入统计（例如丢失年份的命中率）并设定告警阈值。
- 单元测试：为 `generateQuestionsWithContext` 增加单元/表格测试，覆盖：
  - 行以年份开头的情况
  - 以 `1.`/`-` 列表标记开头的情况
  - 空行或异常换行的情况

**保存位置**
- 文档已保存为：
  - `docs/CHANGELOG/knowledge-fixes-2026-01-03.md`

---

如需我将这份变更以 Commit 的形式提交并生成对应 PR，我可以继续：
- 生成并提交 Commit（包含此文档与变更说明），或
- 为 `generateQuestionsWithContext` 添加单元测试并运行 CI（如果您希望我继续，我会继续接下来的步骤）。

若要我继续提交或创建 PR，请回复“提交变更”。