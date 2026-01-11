# aiplusall-kb Product Overview

aiplusall-kb is an LLM-powered document understanding and retrieval framework designed for deep semantic analysis of complex, heterogeneous documents. It follows the RAG (Retrieval-Augmented Generation) paradigm to provide high-quality, context-aware answers.

## Core Features

- **Agent Mode**: ReACT Agent with built-in tools, MCP integration, and web search capabilities
- **Multi-Type Knowledge Bases**: Support for FAQ and document knowledge bases with various import methods
- **Document Processing**: Handles PDFs, Word docs, images, markdown, and more with OCR/caption support
- **Intelligent Retrieval**: Hybrid strategies combining keywords, vectors, and knowledge graphs
- **Knowledge Graph**: Transform documents into semantic relationship networks using Neo4j
- **Multi-Tenant Architecture**: Role-based access control with tenant isolation
- **Legal Domain Specialization**: Enhanced for legal document analysis with compliance features

## Target Use Cases

- Enterprise knowledge management and internal document retrieval
- Legal document analysis and case law research
- Academic research and literature review
- Technical documentation and support systems
- Regulatory compliance and policy analysis

## Architecture

aiplusall-kb uses a modular microservices architecture with:
- Go backend API server
- Python document reader service (gRPC)
- Vue.js frontend
- PostgreSQL with pgvector for data storage
- Redis for caching and task queues
- Optional Neo4j for knowledge graphs
- Optional Minio for file storage