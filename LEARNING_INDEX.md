# 学习文档索引

> **快速导航**：所有学习文档的索引和快速链接

---

## 📚 文档列表

### 基础文档

1. **[README_CN.md](README_CN.md)** - 项目主文档（中文版）
   - 项目概述
   - 系统架构
   - 组件说明

2. **[PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md)** - 项目概述（中文）
   - 项目简介
   - 技术架构
   - 核心流程
   - 学习路径

3. **[LEARNING_GUIDE.md](LEARNING_GUIDE.md)** - 详细学习指南（中文）
   - 完整的流程详解
   - 学习路径
   - 关键技术点

### 分阶段学习文档

#### 第一阶段：理解整体架构

**[STAGE1_LEARNING_GUIDE.md](STAGE1_LEARNING_GUIDE.md)** - 第一阶段学习指南
- **适合人群**：推荐算法新手小白
- **预计时间**：1-2天
- **学习目标**：
  - 理解推荐系统的基本概念
  - 理解项目的整体架构
  - 理解各个组件的作用
- **关键内容**：
  - 什么是推荐系统
  - 两阶段推荐架构
  - 核心组件说明
  - 关键概念解释

#### 第二阶段：理解数据流

**[STAGE2_LEARNING_GUIDE.md](STAGE2_LEARNING_GUIDE.md)** - 第二阶段学习指南
- **适合人群**：已完成第一阶段学习
- **预计时间**：2-3天
- **学习目标**：
  - 追踪一个请求的完整处理流程
  - 理解各阶段的数据转换
  - 理解并行 vs 顺序执行
- **关键内容**：
  - 请求入口（gRPC服务）
  - 管道执行引擎
  - 数据流转换
  - 错误处理机制

#### 第三阶段：深入各组件

**[STAGE3_LEARNING_GUIDE.md](STAGE3_LEARNING_GUIDE.md)** - 第三阶段学习指南
- **适合人群**：已完成第二阶段学习
- **预计时间**：3-5天
- **学习目标**：
  - 理解 Sources（候选源）的实现
  - 理解 Filters（过滤器）的逻辑
  - 理解 Hydrators（增强器）的工作
  - 理解 Scorers（打分器）的计算
- **关键内容**：
  - ThunderSource 和 PhoenixSource
  - 各种过滤器的实现
  - 各种增强器的实现
  - 各种打分器的实现

#### 第四阶段：理解 ML 模型

**[STAGE4_LEARNING_GUIDE.md](STAGE4_LEARNING_GUIDE.md)** - 第四阶段学习指南
- **适合人群**：已完成第三阶段学习，对 Python/JAX 有基本了解
- **预计时间**：5-7天
- **学习目标**：
  - 理解 Two-Tower 检索模型
  - 理解 Transformer 排序模型
  - 理解 Candidate Isolation 机制
  - 理解多动作预测
- **关键内容**：
  - Two-Tower 架构详解
  - Transformer 架构详解
  - Candidate Isolation 实现
  - Hash-Based Embeddings

#### 第五阶段：实践与实验

**[STAGE5_LEARNING_GUIDE.md](STAGE5_LEARNING_GUIDE.md)** - 第五阶段学习指南
- **适合人群**：已完成前四个阶段学习
- **预计时间**：持续（根据实验需求）
- **学习目标**：
  - 运行和测试整个系统
  - 进行性能分析和优化
  - 添加新功能和组件
  - 进行 A/B 测试
- **关键内容**：
  - 环境搭建和运行
  - 实验和修改
  - 性能优化
  - A/B 测试
  - 监控和调试

### Phoenix 组件文档

**[phoenix/README_CN.md](phoenix/README_CN.md)** - Phoenix 组件说明（中文版）
- Phoenix 概述
- 两阶段推荐管道
- Two-Tower 模型
- Transformer 排序模型

---

## 🗺️ 学习路径

### 推荐学习顺序

```
开始
  ↓
第一阶段：理解整体架构（1-2天）
  ├─ 阅读 README_CN.md
  ├─ 阅读 PROJECT_OVERVIEW.md
  └─ 完成 STAGE1_LEARNING_GUIDE.md
  ↓
第二阶段：理解数据流（2-3天）
  ├─ 阅读 LEARNING_GUIDE.md（参考）
  └─ 完成 STAGE2_LEARNING_GUIDE.md
  ↓
第三阶段：深入各组件（3-5天）
  └─ 完成 STAGE3_LEARNING_GUIDE.md
  ↓
第四阶段：理解 ML 模型（5-7天）
  ├─ 阅读 phoenix/README_CN.md
  └─ 完成 STAGE4_LEARNING_GUIDE.md
  ↓
第五阶段：实践与实验（持续）
  └─ 完成 STAGE5_LEARNING_GUIDE.md
  ↓
完成学习！
```

### 快速开始

**如果你是新手**：
1. 从 [STAGE1_LEARNING_GUIDE.md](STAGE1_LEARNING_GUIDE.md) 开始
2. 按照指南逐步学习

**如果你有经验**：
1. 快速浏览 [PROJECT_OVERVIEW.md](PROJECT_OVERVIEW.md)
2. 选择感兴趣的阶段深入学习

---

## 📖 文档使用指南

### 如何阅读文档

1. **按顺序阅读**：
   - 建议按照阶段顺序阅读
   - 每个阶段都有前置要求

2. **结合代码**：
   - 阅读文档时，同时阅读相关代码
   - 使用调试器追踪执行流程

3. **做笔记**：
   - 每个阶段都有学习笔记模板
   - 记录你的理解和疑问

4. **实践练习**：
   - 完成每个阶段的实践练习
   - 验证你的理解

### 文档特点

- ✅ **循序渐进**：从基础到深入
- ✅ **详细全面**：覆盖所有关键点
- ✅ **实践导向**：包含大量实践练习
- ✅ **中文友好**：所有文档都有中文版本

---

## 🔍 快速查找

### 按主题查找

**推荐系统基础**：
- [STAGE1_LEARNING_GUIDE.md](STAGE1_LEARNING_GUIDE.md) - 第一阶段

**数据流和架构**：
- [STAGE2_LEARNING_GUIDE.md](STAGE2_LEARNING_GUIDE.md) - 第二阶段
- [LEARNING_GUIDE.md](LEARNING_GUIDE.md) - 完整流程

**组件实现**：
- [STAGE3_LEARNING_GUIDE.md](STAGE3_LEARNING_GUIDE.md) - 第三阶段

**ML 模型**：
- [STAGE4_LEARNING_GUIDE.md](STAGE4_LEARNING_GUIDE.md) - 第四阶段
- [phoenix/README_CN.md](phoenix/README_CN.md) - Phoenix 文档

**实践和实验**：
- [STAGE5_LEARNING_GUIDE.md](STAGE5_LEARNING_GUIDE.md) - 第五阶段

### 按问题查找

**"推荐系统是什么？"**
→ [STAGE1_LEARNING_GUIDE.md](STAGE1_LEARNING_GUIDE.md) - 第一步

**"数据是如何流动的？"**
→ [STAGE2_LEARNING_GUIDE.md](STAGE2_LEARNING_GUIDE.md) - 完整流程

**"过滤器是如何工作的？"**
→ [STAGE3_LEARNING_GUIDE.md](STAGE3_LEARNING_GUIDE.md) - 第二部分

**"ML 模型是如何工作的？"**
→ [STAGE4_LEARNING_GUIDE.md](STAGE4_LEARNING_GUIDE.md) - 完整内容

**"如何运行和测试代码？"**
→ [STAGE5_LEARNING_GUIDE.md](STAGE5_LEARNING_GUIDE.md) - 第一部分

---

## 💡 学习建议

### 给新手的建议

1. **不要着急**：
   - 每个阶段都需要时间理解
   - 理解透彻比快速完成更重要

2. **多画图**：
   - 画出数据流图
   - 画出架构图
   - 帮助理解和记忆

3. **多实践**：
   - 完成每个阶段的实践练习
   - 修改代码，观察变化

4. **多提问**：
   - 记录不懂的地方
   - 寻求帮助

### 给有经验者的建议

1. **快速浏览**：
   - 快速浏览基础阶段
   - 重点关注感兴趣的领域

2. **深入实践**：
   - 直接进入实践阶段
   - 通过实验加深理解

3. **贡献代码**：
   - 添加新功能
   - 优化性能
   - 分享经验

---

## 📝 学习进度跟踪

### 进度检查清单

- [ ] 第一阶段：理解整体架构
- [ ] 第二阶段：理解数据流
- [ ] 第三阶段：深入各组件
- [ ] 第四阶段：理解 ML 模型
- [ ] 第五阶段：实践与实验

### 学习笔记

建议为每个阶段创建学习笔记：
- 使用每个阶段提供的笔记模板
- 记录你的理解和疑问
- 记录实验结果和发现

---

## 🎯 学习目标总结

完成所有阶段后，你应该能够：

1. ✅ **理解推荐系统**：理解推荐系统的基本概念和架构
2. ✅ **理解数据流**：追踪一个请求的完整处理流程
3. ✅ **理解组件**：理解各个组件的实现和工作原理
4. ✅ **理解 ML 模型**：理解检索和排序模型的架构
5. ✅ **实践能力**：能够运行、修改、优化和扩展系统

---

## 🚀 下一步

1. **开始学习**：从第一阶段开始
2. **选择路径**：根据你的背景选择合适的学习路径
3. **持续学习**：不断深入和实践

**祝你学习顺利！🎉**

---

**最后更新**：2024年

**文档维护**：如有问题或建议，请提交 Issue 或 PR
