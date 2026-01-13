package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ethereum/go-ethereum/common/hexutil"
	pb "github.com/zzz136454872/upgradeable-consensus/pkg/proto"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// 创建测试用的 Coinbase 交易，为指定地址分配初始余额
func createTestCoinbase(targetAddr []byte, amount int64) *types.RawTx {
	fmt.Printf("📦 Creating test Coinbase transaction...\n")
	fmt.Printf("   Target: %s\n", hexutil.Encode(targetAddr))
	fmt.Printf("   Amount: %d\n", amount)

	// Coinbase 输入（标准格式）
	txin := types.TxInput{
		Txid:      [32]byte{},
		Voutput:   -1, // 关键：标记为 coinbase
		Scriptsig: nil,
		Value:     0,
		Address:   []byte{},
		BciType:   0,
	}

	// 目标地址输出
	txout := types.TxOutput{
		Address:  targetAddr,
		Value:    amount,
		ScriptPk: targetAddr,
		LockTime: 7,
		BciType:  1,
		BurnLock: 0,
		Interest: 0,
		Rate:     0,
		Proof:    []byte{},
	}

	tx := &types.RawTx{
		Txid:           [32]byte{},
		TxInput:        []types.TxInput{txin},
		TxOutput:       []types.TxOutput{txout},
		CoinbaseProofs: []types.CoinbaseProof{},
		TransactionFee: 0,
	}

	// 计算并设置交易哈希
	tx.Txid = tx.Hash()

	fmt.Printf("✅ TxID: %s\n", hexutil.Encode(tx.Txid[:]))
	return tx
}

func main() {
	serverAddr := "127.0.0.1:9866"

	fmt.Println("🚀 Test UTXO Creator for BCI System")
	fmt.Println("=====================================\n")

	// 连接到服务器
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("❌ Failed to connect to %s: %v", serverAddr, err)
	}
	defer conn.Close()

	client := pb.NewBciExectorClient(conn)

	// 获取一个可用的 PQC 密钥（尝试多个高度）
	fmt.Println("🔑 Getting PQC key for test address...")
	var targetAddr []byte

	for tryHeight := uint64(1); tryHeight <= 50; tryHeight++ {
		keyreq := &pb.GetPqcKeyRequest{Height: tryHeight}
		keyresp, err := client.GetPqcKey(context.Background(), keyreq)
		if err == nil && keyresp.PublicKey != nil {
			targetAddr = keyresp.PublicKey
			fmt.Printf("✅ Found PQC key at height %d\n", tryHeight)
			fmt.Printf("   Address: %s\n\n", hexutil.Encode(targetAddr))
			break
		}
	}

	if targetAddr == nil {
		fmt.Println("❌ Could not get any PQC key from the node")
		fmt.Println("💡 Make sure the server is running: make run_server")
		return
	}

	// 创建测试 Coinbase
	testAmount := int64(1000000) // 100万单位
	coinbaseTx := createTestCoinbase(targetAddr, testAmount)

	fmt.Println("\n💡 Manual Steps Required:")
	fmt.Println("=====================================")
	fmt.Println("Since there's no direct API to inject test coinbase,")
	fmt.Println("you need to:")
	fmt.Println()
	fmt.Println("Option 1: Wait for mining (Recommended)")
	fmt.Println("  1. Run: make run_server")
	fmt.Println("  2. Wait for 15+ blocks to be mined")
	fmt.Println("  3. Run: go run tests/bci/main.go")
	fmt.Println()
	fmt.Println("Option 2: Modify genesis block")
	fmt.Println("  1. Edit: consensus/pot/chainreader.go")
	fmt.Println("  2. Add test UTXOs to DefaultGenesisBlock()")
	fmt.Println("  3. Delete: data/node-0/ (reset chain)")
	fmt.Println("  4. Restart: make run_server")
	fmt.Println()

	// 显示创建的交易详情
	fmt.Println("📋 Created Transaction Details:")
	fmt.Println("=====================================")
	fmt.Printf("TxID:     %s\n", hexutil.Encode(coinbaseTx.Txid[:]))
	fmt.Printf("IsCoinbase: %v\n", coinbaseTx.IsCoinBase())
	fmt.Printf("Inputs:   %d (Voutput: %d)\n", len(coinbaseTx.TxInput), coinbaseTx.TxInput[0].Voutput)
	fmt.Printf("Outputs:  %d\n", len(coinbaseTx.TxOutput))
	fmt.Printf("  - Address: %s\n", hexutil.Encode(coinbaseTx.TxOutput[0].Address))
	fmt.Printf("  - Value:   %d\n", coinbaseTx.TxOutput[0].Value)
	fmt.Printf("  - BciType: %d\n", coinbaseTx.TxOutput[0].BciType)

	// 保存交易数据（可选）
	txData, _ := coinbaseTx.EncodeToByte()
	fmt.Printf("\nEncoded size: %d bytes\n", len(txData))

	fmt.Println("\n✨ Done! Now run the mining node to create real UTXOs.")
}
