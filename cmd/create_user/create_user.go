package main

import (
	"blockchain-go/pkg/wallet"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Cách dùng: cli create-user --name alice")
		return
	}

	switch os.Args[1] {
	case "create-user":
		fs := flag.NewFlagSet("create-user", flag.ExitOnError)
		name := fs.String("name", "", "user name (must)")
		fs.Parse(os.Args[2:])

		if *name == "" {
			fmt.Println("❌ You must provide --name")
			return
		}

		createUser(*name)

	default:
		fmt.Println("Command Invalid. Use: cli create-user --name alice")
	}
}

func createUser(name string) {
	relPath := filepath.Join("wallets", name+".json")
	absPath, _ := filepath.Abs(relPath)

	if wallet.WalletExists(relPath) {
		fmt.Printf("❌ Account '%s' already exists at:\n📁 %s\n", name, absPath)
		return
	}

	w, err := wallet.CreateWallet()
	if err != nil {
		fmt.Println("❌ Error creating account:", err)
		return
	}

	err = w.SaveToFile(relPath)
	if err != nil {
		fmt.Println("❌ Error saving account:", err)
		return
	}

	fmt.Println("✅ Account saved successfully!")
	fmt.Printf("👤 Name: %s\n", name)
	fmt.Printf("🏦 Address: %s\n", w.Address)
	fmt.Printf("📁 File saved at: %s\n", absPath)
}
