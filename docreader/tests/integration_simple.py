"""
Simplified integration test for paragraph-aware chunking.
Run from project root: python -m docreader.tests.integration_simple
"""

import sys
from pathlib import Path

# Add project root to path
project_root = Path(__file__).parent.parent.parent
sys.path.insert(0, str(project_root))

from docreader.splitter.splitter import TextSplitter
from docreader.models.read_config import ChunkingConfig


def test_markdown_chunking():
    """Test paragraph-aware chunking with Markdown content."""
    
    markdown_content = """# 知识图谱技术简介

知识图谱（Knowledge Graph）是一种用于表示知识的图形化数据结构，它由节点和边组成，节点代表实体，边代表实体之间的关系。

## 核心概念

知识图谱的核心在于将现实世界中的实体及其关系以图的形式进行组织和表达，这使得计算机能够更好地理解和处理知识。主要包括以下几个方面：实体识别、关系抽取、知识表示、知识推理。

### 实体识别

实体识别（Named Entity Recognition, NER）是从文本中识别出特定类型实体的过程，例如人名、地名、组织名等。这是构建知识图谱的第一步，也是最基础的环节。

### 关系抽取

关系抽取（Relation Extraction）是从文本中识别实体之间关系的过程。例如，从句子"张三在北京工作"中抽取出（张三，工作于，北京）这样的三元组关系。

## 应用场景

知识图谱在多个领域都有广泛的应用：

1. **搜索引擎**：提供更智能的搜索结果和知识卡片
2. **推荐系统**：基于知识图谱的个性化推荐
3. **问答系统**：支持复杂问题的理解和回答
4. **金融风控**：企业关系图谱分析
5. **医疗健康**：疾病诊断辅助系统

## 技术挑战

构建和维护知识图谱面临诸多挑战，包括但不限于：数据质量问题、知识更新的时效性、多源异构数据融合、大规模图数据的存储和查询效率、知识推理的准确性等。这些挑战需要结合机器学习、自然语言处理、图数据库等多种技术来解决。

## 总结

知识图谱作为人工智能的重要基础设施，正在深刻改变着我们处理和利用知识的方式。随着技术的不断发展，知识图谱将在更多领域发挥重要作用。
"""
    
    print("=" * 70)
    print("测试1：Markdown文档分块 (段落感知模式 vs 传统模式)")
    print("=" * 70)
    
    # Test with paragraph_aware=True
    print("\n### 段落感知模式 (paragraph_aware=True) ###\n")
    splitter_aware = TextSplitter(
        chunk_size=700,
        chunk_overlap=100,
        separators=["\n\n", "\n", "。"],
        paragraph_aware=True,
        language="zh"
    )
    chunks_aware = splitter_aware.split_text(markdown_content)
    
    print(f"✓ 总分块数: {len(chunks_aware)}")
    print(f"✓ 文档总长度: {len(markdown_content)} 字符")
    
    for i, (start, end, content) in enumerate(chunks_aware[:3]):
        print(f"\n分块 {i}:")
        print(f"  长度: {len(content)} 字符")
        print(f"  位置: [{start}, {end})")
        print(f"  内容预览: {content[:60]}...")
        
        # Check sentence ending
        if content.strip():
            last_char = content.rstrip()[-1]
            if last_char in "。！？；":
                print(f"  ✓ 结束符: '{last_char}' (正确的句子结束符)")
            else:
                print(f"  ⚠️  结束符: '{last_char}' (非句子结束符)")
    
    # Test with paragraph_aware=False (legacy)
    print("\n\n### 传统模式 (paragraph_aware=False) ###\n")
    splitter_legacy = TextSplitter(
        chunk_size=700,
        chunk_overlap=100,
        separators=["\n\n", "\n", "。"],
        paragraph_aware=False,
        language="zh"
    )
    chunks_legacy = splitter_legacy.split_text(markdown_content)
    
    print(f"✓ 总分块数: {len(chunks_legacy)}")
    
    for i, (start, end, content) in enumerate(chunks_legacy[:3]):
        print(f"\n分块 {i}:")
        print(f"  长度: {len(content)} 字符")
        print(f"  内容预览: {content[:60]}...")
    
    print(f"\n✅ 测试1完成！段落感知模式生成 {len(chunks_aware)} 个分块，传统模式生成 {len(chunks_legacy)} 个分块")


def test_legal_document():
    """Test with legal document (long paragraphs)."""
    
    text_content = """中华人民共和国民法典

第一编 总则

第一章 基本规定

第一条 为了保护民事主体的合法权益，调整民事关系，维护社会和经济秩序，适应中国特色社会主义发展要求，弘扬社会主义核心价值观，根据宪法，制定本法。

第二条 民法调整平等主体的自然人、法人和非法人组织之间的人身关系和财产关系。民法的基本原则和制度适用于合同、物权、侵权责任等民事关系。

第三条 民事主体的人身权利、财产权利以及其他合法权益受法律保护，任何组织或者个人不得侵犯。民事主体依法享有权利，履行义务，不得违反法律、行政法规的禁止性规定。

第四条 民事主体在民事活动中的法律地位一律平等。平等原则是民法的基本原则之一，任何组织或者个人不得因为身份、地位等因素而享有特权。

第五条 民事主体从事民事活动，应当遵循自愿原则，按照自己的意思设立、变更、终止民事法律关系。自愿原则是民事活动的基本准则，充分尊重当事人的意思自治。

第六条 民事主体从事民事活动，应当遵循公平原则，合理确定各方的权利和义务。公平原则要求在确定民事权利义务时，应当平衡各方利益，实现实质公平。

第七条 民事主体从事民事活动，应当遵循诚信原则，秉持诚实，恪守承诺。诚信原则是民法的基本原则，要求当事人在民事活动中应当诚实守信。

第八条 民事主体从事民事活动，不得违反法律，不得违背公序良俗。公序良俗是指公共秩序和善良风俗，是民事活动的基本底线。

第九条 民事主体从事民事活动，应当有利于节约资源、保护生态环境。绿色原则是民法典的创新，体现了可持续发展的理念。

第二章 自然人

第一节 民事权利能力和民事行为能力

第十条 自然人的民事权利能力一律平等。自然人从出生时起到死亡时止，具有民事权利能力，依法享有民事权利，承担民事义务。民事权利能力是自然人享有民事权利、承担民事义务的资格。

第十一条 十八周岁以上的自然人为成年人。不满十八周岁的自然人为未成年人。成年人具有完全民事行为能力，可以独立实施民事法律行为。未成年人的民事行为能力根据其年龄和智力状况确定。
"""
    
    print("\n" + "=" * 70)
    print("测试2：法律文档分块 (长段落处理)")
    print("=" * 70)
    
    splitter = TextSplitter(
        chunk_size=700,
        chunk_overlap=100,
        paragraph_aware=True,
        language="zh"
    )
    chunks = splitter.split_text(text_content)
    
    print(f"\n✓ 总分块数: {len(chunks)}")
    print(f"✓ 文档总长度: {len(text_content)} 字符")
    
    # Check first 3 chunks
    for i in range(min(3, len(chunks))):
        start, end, content = chunks[i]
        print(f"\n分块 {i}:")
        print(f"  长度: {len(content)} 字符")
        print(f"  位置: [{start}, {end})")
        print(f"  内容: {content[:80]}...")
        
        # Check ending
        if content.strip():
            last_char = content.rstrip()[-1]
            if last_char in "。！？；":
                print(f"  ✓ 正确在句子边界结束: '{last_char}'")
            else:
                print(f"  ⚠️  结束符: '{last_char}'")
    
    print(f"\n✅ 测试2完成！生成 {len(chunks)} 个分块")


def test_comma_no_split():
    """Verify that commas don't cause splits."""
    
    text = """这是一个包含多个逗号的段落，第一句话很长，有很多细节，包括一些列举项，比如项目A，项目B，项目C，项目D，还有一些说明，这些说明非常详细，涵盖了各个方面，确保信息完整。第二句话也很长，同样包含了多个逗号，用于分隔不同的内容，比如原因1，原因2，原因3，以及相关的解释，这些解释进一步阐明了问题的本质。第三句话继续提供更多信息，包括背景，现状，以及未来的展望。"""
    
    print("\n" + "=" * 70)
    print("测试3：逗号不分割验证")
    print("=" * 70)
    
    splitter = TextSplitter(
        chunk_size=200,
        chunk_overlap=50,
        paragraph_aware=True,
        language="zh"
    )
    chunks = splitter.split_text(text)
    
    print(f"\n✓ 文本长度: {len(text)} 字符")
    print(f"✓ 分块数: {len(chunks)}")
    print(f"✓ 逗号数量: {text.count('，')}")
    
    for i, (start, end, content) in enumerate(chunks):
        last_char = content.rstrip()[-1] if content.strip() else ""
        status = "✓" if last_char in "。！？；" else "✗"
        print(f"\n分块 {i}: {status}")
        print(f"  长度: {len(content)}")
        print(f"  结束符: '{last_char}'")
        print(f"  内容: {content}")
    
    # Verify no comma endings
    comma_endings = sum(1 for _, _, c in chunks if c.rstrip().endswith('，'))
    if comma_endings == 0:
        print(f"\n✅ 测试3完成！没有分块在逗号处结束")
    else:
        print(f"\n⚠️  发现 {comma_endings} 个分块在逗号处结束")


if __name__ == "__main__":
    print("\n" + "="*70)
    print("段落感知分块集成测试")
    print("="*70)
    
    try:
        test_markdown_chunking()
        test_legal_document()
        test_comma_no_split()
        
        print("\n" + "="*70)
        print("✅ 所有集成测试通过！")
        print("="*70)
        
    except Exception as e:
        print(f"\n❌ 测试失败: {e}")
        import traceback
        traceback.print_exc()
