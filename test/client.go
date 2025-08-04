package main

import (
	"context"
	"fmt"
	"time"

	pb "CraneNetWeak/generated/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
)

func main() {
	// 用 unix socket 连接
	conn, err := grpc.NewClient(
		"unix:///var/craned/crane_test.sock",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		fmt.Printf("Failed to dial: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	client := pb.NewCraneTestClient(conn)

	// 构造 NetWeakStartRequest
	startReq := &pb.NetWeakStartRequest{
		NodeList: []string{"h1", "h2"}, // 替换为你的节点名
		Option:   "",                   // 或留空让服务端自动生成
	}

	// 调用 NetWeakStart
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	startResp, err := client.NetWeakStart(ctx, startReq)
	if err != nil {
		fmt.Printf("NetWeakStart error: %v\n", err)
		return
	}
	fmt.Printf("NetWeakStart reply: ok=%v, msg=%s\n", startResp.Ok, startResp.Msg)

	// 你可以等待一会儿，再调用停止
	time.Sleep(200 * time.Second)

	// 调用 NetWeakStop
	stopReq := &pb.Empty{}
	stopResp, err := client.NetWeakStop(ctx, stopReq)
	if err != nil {
		fmt.Printf("NetWeakStop error: %v\n", err)
		return
	}
	fmt.Printf("NetWeakStop reply: ok=%v, msg=%s\n", stopResp.Ok, stopResp.Msg)
}
