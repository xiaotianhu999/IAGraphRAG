# 故障排查指南

## LLM循环输出问题

### 现象
对话响应中出现循环重复文本，如："the full text of the full text of the full text of..."

### 可能原因
1. LLM模型陷入重复生成循环
2. 上下文tokens过多导致模型行为异常
3. 提示词中的重复短语触发模型异常

### 排查步骤

#### 1. 检查LLM模型配置
```bash
# 查看当前使用的模型
cat .env | grep MODEL

# 检查模型参数
docker logs aiplusall-kb-app-dev 2>&1 | grep -i "model\|temperature\|max_tokens"
```

#### 2. 清空对话历史
如果是上下文过长导致的，清空当前会话重新开始：
- 在前端点击"新建对话"
- 或调用API删除当前会话

#### 3. 调整模型参数
在`.env`或配置文件中调整以下参数：

```env
# 降低temperature减少随机性
TEMPERATURE=0.7  # 默认值，可降至0.3-0.5

# 增加repetition_penalty防止重复
REPETITION_PENALTY=1.2  # Ollama模型支持

# 限制最大token数
MAX_TOKENS=2000
```

#### 4. 检查Agent配置
```bash
# 查看Agent最大循环次数
grep -r "MaxIterations" internal/agent/
```

### 临时解决方案

1. **重启应用**
   ```bash
   ./scripts/dev.sh restart app
   ```

2. **切换模型**
   如果使用Ollama，尝试切换到更稳定的模型：
   ```bash
   # 在前端模型设置中选择其他模型
   # 如：qwen2.5、llama3.1等
   ```

3. **清理数据库缓存**
   ```bash
   # 清理Redis缓存
   docker exec -i aiplusall-kb-redis-dev redis-cli FLUSHALL
   ```

### 代码级修复（如果需要）

如果问题持续，可以添加重复检测：

```go
// 在 internal/agent/engine.go 的 streamThinkingToEventBus 方法中添加
var lastChunks []string
const maxRepeatCheck = 5

func checkRepetition(content string, lastChunks []string) bool {
    if len(lastChunks) < maxRepeatCheck {
        return false
    }
    // 检查最近5个chunk是否完全相同
    for i := 1; i < maxRepeatCheck; i++ {
        if lastChunks[i] != lastChunks[0] {
            return false
        }
    }
    return true
}
```

### 预防措施

1. **定期监控LLM响应**
   ```bash
   # 监控异常长的响应
   docker logs aiplusall-kb-app-dev -f | grep -E "content_len|answer_len"
   ```

2. **设置合理的超时**
   在Agent配置中设置每轮最大执行时间

3. **使用更稳定的模型**
   推荐使用经过充分测试的模型版本

### 相关问题

- chunk删除功能已修复（commit e2d00de）
- 数据一致性问题已解决
- 本问题与chunk删除修改无关
