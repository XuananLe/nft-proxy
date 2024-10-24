package main

import (
	"context"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/joho/godotenv"
	"log"
	"os"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	// Consider having a config object that holds all the enviroment variable
	client := rpc.New(os.Getenv("RPC_URL"))
	resp, err := client.GetAccountInfo(context.TODO(), solana.MustPublicKeyFromBase58("9FmQSV65qhKC4uDe1U2npGxAeMD7Ym2AQgBtkKyjKaax"))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%s", resp.Value.Data.GetBinary())

	pResp, err := client.GetProgramAccountsWithOpts(context.TODO(), solana.MustPublicKeyFromBase58("CoREENxT6tW1HoK8ypY1SxRMZTcVPm7R94rH4PZNhX7d"), &rpc.GetProgramAccountsOpts{
		Filters: []rpc.RPCFilter{
			{DataSize: 1 + 32 + 33 + 4 + 4 + 1},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	for _, r := range pResp {
		log.Printf("%s - Size: %v", r.Pubkey, len(r.Account.Data.GetBinary()))
	}

}
