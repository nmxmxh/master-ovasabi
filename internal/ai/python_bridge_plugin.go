package ai

import (
	"context"
	"fmt"
	"time"

	aipb "github.com/nmxmxh/master-ovasabi/api/protos/ai/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// PythonBridgePlugin implements Plugin, EmbeddingPlugin, and LLMPlugin by forwarding calls to a Python AI gRPC service.
type PythonBridgePlugin struct {
	Endpoint string // gRPC address of the Python AI service, e.g. "localhost:50051"
	Client   aipb.AIServiceClient
	Conn     *grpc.ClientConn
	Info     PluginInfo
}

func NewPythonBridgePlugin(endpoint string) *PythonBridgePlugin {
	// Establish gRPC connection using modern credentials (NewClient)
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Python AI gRPC service: %v", err))
	}
	client := aipb.NewAIServiceClient(conn)
	return &PythonBridgePlugin{
		Endpoint: endpoint,
		Client:   client,
		Conn:     conn,
		Info: PluginInfo{
			Name:    "PythonBridgePlugin",
			Version: "1.0",
			Author:  "OVASABI",
		},
	}
}

func (p *PythonBridgePlugin) Infer(input []byte) ([]byte, error) {
	// Use ProcessContent for inference (assuming text input)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := &aipb.EnrichmentRequest{
		Content: &aipb.EnrichmentRequest_RawData{RawData: input},
	}
	stream, err := p.Client.ProcessContent(ctx)
	if err != nil {
		return nil, err
	}
	if err := stream.Send(req); err != nil {
		return nil, err
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}
	// Return the marshaled response (could be improved to extract summary, etc.)
	return resp.ProtoReflect().Interface().(interface{ MarshalVT() ([]byte, error) }).MarshalVT()
}

func (p *PythonBridgePlugin) Summarize(input []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := &aipb.EnrichmentRequest{
		Content: &aipb.EnrichmentRequest_RawData{RawData: input},
	}
	stream, err := p.Client.ProcessContent(ctx)
	if err != nil {
		return "", err
	}
	if err := stream.Send(req); err != nil {
		return "", err
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", err
	}
	if resp.GetText() != nil {
		return resp.GetText().GetSummary(), nil
	}
	return "", fmt.Errorf("no summary in response")
}

func (p *PythonBridgePlugin) Embed(input []byte) ([]float32, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req := &aipb.EnrichmentRequest{
		Content: &aipb.EnrichmentRequest_RawData{RawData: input},
	}
	resp, err := p.Client.GenerateEmbeddings(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.GetValues(), nil
}

func (p *PythonBridgePlugin) Metadata() PluginInfo {
	return p.Info
}
