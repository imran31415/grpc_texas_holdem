package client

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"google.golang.org/grpc"
	pb "grpc_texas_holdem/poker/protobufs"
)

const (
	address     = "localhost:50051"
	defaultName = "Dumbo, you didnt specify a name!"
)

//  go run grpc_texas_holdem/pokerproject/poker_client/main.go
func Run() {
	conn, c := CreateConnectionClient()
	defer conn.Close()

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := CreateOnePlayer(c, ctx, name, rand.Int63(), 0)

	if err != nil {
		log.Fatalf("could not create player: %v", err)
	}
	log.Printf("Greeting: %s, your ID is %d and your chipcount is %d", r.GetName(), r.GetId(), r.GetChips())
}

func CreateOnePlayer(client pb.PokerClient, ctx context.Context, name string, id int64, chips int64) (*pb.Player, error) {
	r, err := client.CreatePlayer(ctx, &pb.Player{Name: name, Id: id, Chips: chips})
	if err != nil {
		return nil, err
	}
	return r, nil

}

func CreateConnectionClient() (*grpc.ClientConn, pb.PokerClient) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	c := pb.NewPokerClient(conn)
	return conn, c
}
