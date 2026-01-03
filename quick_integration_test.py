"""快速集成测试 - 验证段落感知分块功能"""

import sys
sys.path.insert(0, "d:/workdir/ai/code/WeKnora")

from docreader.splitter.splitter import TextSplitter

# 测试文本
text = """# 知识图谱简介

知识图谱是一种表示知识的图形化数据结构。它由节点和边组成，节点代表实体，边代表关系。

## 核心概念

知识图谱的核心在于将现实世界中的实体及其关系以图的形式进行组织和表达。主要包括：实体识别、关系抽取、知识表示、知识推理。

## 应用场景

知识图谱应用广泛：搜索引擎、推荐系统、问答系统、金融风控、医疗健康等领域。

## 技术挑战

面临的挑战包括：数据质量问题、知识更新时效性、多源数据融合、存储查询效率、推理准确性等。需要结合机器学习、自然语言处理等技术解决。
"""

print("="*70)
print("集成测试：段落感知分块")
print("="*70)

# 段落感知模式
print("\n【段落感知模式】")
splitter1 = TextSplitter(
    chunk_size=200,
    chunk_overlap=50,
    paragraph_aware=True,
    language="zh"
)
chunks1 = splitter1.split_text(text)

print(f"总分块数: {len(chunks1)}")
for i, (start, end, content) in enumerate(chunks1):
    last_char = content.rstrip()[-1] if content.strip() else ""
    print(f"\n分块{i} (长度{len(content)}字符, 结束'{last_char}'):")
    print(f"  {content[:60]}...")

# 传统模式
print("\n\n【传统模式】")
splitter2 = TextSplitter(
    chunk_size=200,
    chunk_overlap=50,
    paragraph_aware=False,
    language="zh"
)
chunks2 = splitter2.split_text(text)

print(f"总分块数: {len(chunks2)}")
for i, (start, end, content) in enumerate(chunks2):
    print(f"\n分块{i} (长度{len(content)}字符):")
    print(f"  {content[:60]}...")

print("\n" + "="*70)
print(f"✅ 测试完成! 段落感知: {len(chunks1)}块, 传统模式: {len(chunks2)}块")
print("="*70)
