---
title: Experimental Local AI Setup for Pattern Systems
filename: experimental_local_ai.mb
---

## 🧪 Experimental Local AI with Go + gRPC + Ollama/DeepSeek

This document outlines how to set up a local AI-powered pattern generation and retrieval system using Go (gRPC), Ollama/DeepSeek (LLMs), Amadeus (pattern engine), and a knowledge graph (e.g., Nexus). The architecture supports retrieval-augmented generation (RAG), with dynamic context generation and pattern interaction.

---

## 🔧 System Overview

- **Local LLM**: Ollama / DeepSeek (via HTTP API)
- **gRPC API**: Go microservice exposing endpoints to frontend/backend
- **Knowledge Graph/Database**: Nexus (Postgres/Redis)
- **Amadeus**: Pattern manager and validator
- **Embedding Layer**: Sentence Transformers / BGE
- **Vector DB**: Chroma / FAISS

---

## 📂 Directory Structure

```
experimental_local_ai/
├── go-service/
│   ├── main.go
│   ├── server.go
│   ├── proto/
│   │   └── ai.proto
│   └── amadeus/
│       └── engine.go
├── vector-store/
│   └── embed_and_store.py
├── prompts/
│   └── context_template.txt
├── Makefile
└── README.md
```

---

## 🏗️ Setting Up the Vector Store

**Dependencies**:
- Python
- sentence-transformers
- chromadb

**`vector-store/embed_and_store.py`**:
```python
from sentence_transformers import SentenceTransformer
import chromadb

model = SentenceTransformer("all-MiniLM-L6-v2")
db = chromadb.Client()

patterns = ["User Onboarding", "Wallet Relationship"]
embeddings = model.encode(patterns)

for doc, vec in zip(patterns, embeddings):
    db.add(documents=[doc], embeddings=[vec])
```

Run:
```bash
python3 vector-store/embed_and_store.py
```

---

## 🌐 Go gRPC Setup

### `proto/ai.proto`
```protobuf
syntax = "proto3";

service AIService {
  rpc QueryAI (AIRequest) returns (AIResponse);
}

message AIRequest {
  string query = 1;
}

message AIResponse {
  string result = 1;
}
```

### `main.go`
```go
package main

import (
  "log"
  "net"
  pb "./proto"
  "google.golang.org/grpc"
)

func main() {
  lis, err := net.Listen("tcp", ":50051")
  if err != nil {
    log.Fatalf("Failed to listen: %v", err)
  }
  s := grpc.NewServer()
  pb.RegisterAIServiceServer(s, &server{})
  log.Println("gRPC server running at :50051")
  s.Serve(lis)
}
```

### `server.go`
```go
package main

import (
  "context"
  "fmt"
  pb "./proto"
  "net/http"
  "bytes"
  "io/ioutil"
)

type server struct {
  pb.UnimplementedAIServiceServer
}

func (s *server) QueryAI(ctx context.Context, req *pb.AIRequest) (*pb.AIResponse, error) {
  prompt := fmt.Sprintf("Using context: ..., answer: %s", req.Query)

  payload := []byte(fmt.Sprintf(`{"model":"llama3", "prompt":"%s", "stream":false}`, prompt))
  res, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(payload))
  if err != nil {
    return nil, err
  }
  defer res.Body.Close()
  body, _ := ioutil.ReadAll(res.Body)

  return &pb.AIResponse{Result: string(body)}, nil
}
```

---

## 🧪 Amadeus Integration (Sample)

**`amadeus/engine.go`**:
```go
package amadeus

func ValidatePattern(output string) bool {
  // Apply regex or syntax checking on LLM output
  return true
}
```

Hook into the `QueryAI` response to pipe result into Amadeus:
```go
if !amadeus.ValidatePattern(llmResponse) {
    return nil, fmt.Errorf("Invalid pattern response")
}
```

---

## 🧰 Makefile

```makefile
default: run

run:
	go run main.go server.go

proto:
	protoc --go_out=. --go-grpc_out=. proto/ai.proto

embed:
	python3 vector-store/embed_and_store.py
```

---

## 📄 README.md (Usage)

### 1. Generate Protobufs
```bash
make proto
```

### 2. Run Vector Embedding
```bash
make embed
```

### 3. Start gRPC Server
```bash
make run
```

### 4. gRPC Client Request
```go
// Use any gRPC client to call QueryAI with a query string
```

---

## 🚀 Future Work
- Automatic feedback loops into Amadeus
- Frontend web UI
- Pattern simulation based on retrieved context
- Optional: LangChain or LlamaIndex integration

---

End of `experimental_local_ai.mb`