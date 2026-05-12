package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"MangaHub/proto/mangahubpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "get":
		runGet(os.Args[2:])
	case "search":
		runSearch(os.Args[2:])
	case "progress":
		runProgress(os.Args[2:])
	case "profile":
		runProfile(os.Args[2:])
	case "library":
		runLibrary(os.Args[2:])
	default:
		usage()
	}
}

func usage() {
	fmt.Println("Usage:")
	fmt.Println("  grpc-client get --id <manga_id> [--addr <host:port>]")
	fmt.Println("  grpc-client search --q <query> [--author <author>] [--genre <genre>] [--status <status>] [--limit <n>] [--offset <n>] [--addr <host:port>]")
	fmt.Println("  grpc-client progress --manga <id> --chapter <n> [--list <name>] [--status <status>] --token <jwt> [--addr <host:port>]")
	fmt.Println("  grpc-client profile --token <jwt> [--addr <host:port>]")
	fmt.Println("  grpc-client library [--list <name>] --token <jwt> [--addr <host:port>]")
}

// dial creates a gRPC client connection to the specified address using insecure credentials
func dial(addr string) (*grpc.ClientConn, error) {
	return grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func withAuth(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewOutgoingContext(ctx, md)
}

func marshalProto(msg proto.Message) ([]byte, error) {
	return protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(msg)
}

func printProto(label string, msg proto.Message) {
	if label != "" {
		fmt.Println(label)
	}
	data, err := protojson.MarshalOptions{Indent: "  ", EmitUnpopulated: true}.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling Proto: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func runGet(args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	addr := fs.String("addr", getenvDefault("GRPC_ADDR", "localhost:50051"), "gRPC address")
	id := fs.String("id", "", "manga id")
	_ = fs.Parse(args)

	if *id == "" {
		log.Fatal("--id is required")
	}

	conn, err := dial(*addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := mangahubpb.NewMangaServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetManga(ctx, &mangahubpb.GetMangaRequest{Id: *id})
	if err != nil {
		log.Fatal(err)
	}
	printProto("Manga:", resp)
}

func runSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	addr := fs.String("addr", getenvDefault("GRPC_ADDR", "localhost:50051"), "gRPC address")
	query := fs.String("q", "", "query")
	author := fs.String("author", "", "author")
	genre := fs.String("genre", "", "genre")
	status := fs.String("status", "", "status")
	limit := fs.Int("limit", 20, "limit")
	offset := fs.Int("offset", 0, "offset")
	_ = fs.Parse(args)

	conn, err := dial(*addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := mangahubpb.NewMangaServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.SearchManga(ctx, &mangahubpb.SearchRequest{
		Query:  *query,
		Author: *author,
		Genre:  *genre,
		Status: *status,
		Limit:  int32(*limit),
		Offset: int32(*offset),
	})
	if err != nil {
		log.Fatal(err)
	}
	printProto("Search response:", resp)
}

func runProgress(args []string) {
	fs := flag.NewFlagSet("progress", flag.ExitOnError)
	addr := fs.String("addr", getenvDefault("GRPC_ADDR", "localhost:50051"), "gRPC address")
	token := fs.String("token", getenvDefault("GRPC_TOKEN", ""), "JWT token")
	mangaID := fs.String("manga", "", "manga id")
	listName := fs.String("list", "", "list name")
	status := fs.String("status", "reading", "status")
	chapter := fs.Int("chapter", 1, "current chapter")
	_ = fs.Parse(args)

	if *mangaID == "" {
		log.Fatal("--manga is required")
	}
	if *token == "" {
		log.Fatal("--token is required")
	}

	conn, err := dial(*addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := mangahubpb.NewMangaServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = withAuth(ctx, *token)
	resp, err := client.UpdateProgress(ctx, &mangahubpb.ProgressRequest{
		MangaId:        *mangaID,
		ListName:       *listName,
		Status:         *status,
		CurrentChapter: int32(*chapter),
	})
	if err != nil {
		log.Fatal(err)
	}
	printProto("Progress response:", resp)
}

func runProfile(args []string) {
	fs := flag.NewFlagSet("profile", flag.ExitOnError)
	addr := fs.String("addr", getenvDefault("GRPC_ADDR", "localhost:50051"), "gRPC address")
	token := fs.String("token", getenvDefault("GRPC_TOKEN", ""), "JWT token")
	_ = fs.Parse(args)

	if *token == "" {
		log.Fatal("--token is required")
	}

	conn, err := dial(*addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := mangahubpb.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = withAuth(ctx, *token)
	resp, err := client.GetProfile(ctx, &emptypb.Empty{})
	if err != nil {
		log.Fatal(err)
	}
	printProto("Profile:", resp)
}

func runLibrary(args []string) {
	fs := flag.NewFlagSet("library", flag.ExitOnError)
	addr := fs.String("addr", getenvDefault("GRPC_ADDR", "localhost:50051"), "gRPC address")
	token := fs.String("token", getenvDefault("GRPC_TOKEN", ""), "JWT token")
	listName := fs.String("list", "", "list name")
	_ = fs.Parse(args)

	if *token == "" {
		log.Fatal("--token is required")
	}

	conn, err := dial(*addr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := mangahubpb.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = withAuth(ctx, *token)
	resp, err := client.GetLibrary(ctx, &mangahubpb.GetLibraryRequest{ListName: *listName})
	if err != nil {
		log.Fatal(err)
	}
	printProto("Library:", resp)
}

func getenvDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
