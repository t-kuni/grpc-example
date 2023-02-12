package main

import (
	"context"
	"fmt"
	"github.com/t-kuni/grpc-example/grpc/chat"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

func main() {
	port := 30000

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	serverAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	client := chat.NewChatClient(conn)

	ctx := context.Background()

	var name string
	fmt.Print("Name: ")
	_, err = fmt.Scan(&name)
	if err != nil {
		log.Fatal(err)
	}

	profile := &chat.Profile{
		Name:   name,
		Age:    10,
		Gender: chat.Gender_GENDER_MAN,
	}
	user, err := client.Join(ctx, profile)
	if err != nil {
		log.Fatal(err)
	}

	for {
		var commentBody string
		fmt.Print("Comment: ")
		_, err = fmt.Scan(&commentBody)
		if err != nil {
			log.Fatal(err)
		}

		comment := &chat.Comment{
			Body:      commentBody,
			Commenter: user,
		}

		_, err = client.SendComment(ctx, comment)
		if err != nil {
			log.Fatal(err)
		}
	}
}
