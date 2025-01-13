package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtype "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zzz136454872/upgradeable-consensus/consensus/pot"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"google.golang.org/grpc"
)

// 连接到gRPC服务器
func connectToServer(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	return conn, nil
}

// 创建客户端并调用服务
func main() {
	//假设服务器地址为localhost:50051
	serverAddr := "127.0.0.1:9866"

	// 连接到服务器
	conn, err := connectToServer(serverAddr)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// 创建客户端
	client := pb.NewDciExectorClient(conn)

	for true {

		// str := "0x13017a8cdc8a5a3929fdd814c925c9db2ff9875fc0b9fd30250f9126af04d5d1decfa524d9226d5f529c0dc708dc3e2f0dcb254c75848f4f16f972712231602f04165df4a72bc8bceaf4effcc62dc59461eba9801cb66c99a9666553f1ba42d1"
		// addr, err := hexutil.Decode(str)
		// //fmt.Println(addr)
		// str1 := "0xd9a26bd283cc41e6bbefea02bbb646e4f4055ab0bd1bf040f485a64f1cc1915f"
		// str2 := "0x4479d3eb7ced7d6da57cfda3897d3437fb52cb7a6805ae1935905a77dec23c67"
		// blockhash, err := hexutil.Decode(str1)
		// txhash, err := hexutil.Decode(str2)
		// dcireward := &pb.DciReward{
		// 	Address: addr,
		// 	Amount:  100,
		// 	ChainID: 1,
		// 	DciProof: &pb.DciProof{
		// 		Height:    37,
		// 		BlockHash: blockhash,
		// 		TxHash:    txhash,
		// 	},
		// }

		// req := &pb.SendDciRequest{DciReward: []*pb.DciReward{dcireward}}

		// resp, err := client.SendDci(context.Background(), req)
		// if err != nil {
		// 	fmt.Println(err)
		// }
		// fmt.Printf("Response: %d\n", resp.IsSuccess)
		// break

		height := uint64(10)
		keyreq := &pb.GetPqcKeyRequest{
			Height: height,
		}
		keyresp, err := client.GetPqcKey(context.Background(), keyreq)
		if err != nil {
			fmt.Println(err)
			return
		}
		addr := keyresp.PublicKey

		// str := "0x5f41dce11f9b9a19b92c6dad6d2b8b8f00000000000000007695cd781cde492f5254a88066024237129eb549b0864c8f"
		// addr, err := hexutil.Decode(str)
		// //fmt.Println(addr)

		req := &pb.GetBalanceRequest{
			Address: addr,
		}

		resp, err := client.GetBalance(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(resp.Balance)

		fmt.Println(len(resp.GetUtxos()))
		fmt.Println(resp.GetBalance())
		utxo := resp.Utxos[0]
		utxoOutput := types.ToTxOutput(utxo.GetTxOutput())
		amount := utxoOutput.Value

		utxokey := fmt.Sprintf("%s:%d", hexutil.Encode(utxo.GetTxid()), utxo.GetVoutput())
		fmt.Println(utxokey)
		txid := resp.GetUtxos()[0].GetTxid()
		pqckey := &crypto.PqcKey{
			Privkey: keyresp.SecretKey,
			Pubkey:  keyresp.PublicKey,
			Scheme:  crypto.PqcScheme,
		}
		times := time.Now()

		sig, err := pqckey.Sign([]byte(utxokey))

		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(float64(time.Since(times)) / float64(time.Millisecond))
		fmt.Println(utxokey)
		time2 := time.Now()
		flag, _ := crypto.VerifySig([]byte(utxokey), sig, pqckey.PublicKeyBytes())
		fmt.Println(float64(time.Since(time2)) / float64(time.Millisecond))
		fmt.Println(flag)
		fmt.Println(len(sig))

		txinput := types.TxInput{
			Txid:      crypto.Convert(txid),
			Voutput:   resp.GetUtxos()[0].GetVoutput(),
			Scriptsig: sig,
			Value:     amount,
			BciType:   utxoOutput.BciType,
			Address:   addr,
		}

		// // txoutput := types.TxOutput{
		// // 	Address:  addr,
		// // 	Value:    amount,
		// // 	BurnLock: 1000 + pot.TenYears + 100,
		// // 	ScriptPk: addr,
		// // 	Proof:    addr,
		// // 	LockTime: 7,
		// // 	Rate:     pot.TenYearRate,
		// // }
		// // tx := types.RawTx{
		// // 	Txid:           [32]byte{},
		// // 	TransactionFee: 0,
		// // 	TxInput:        []types.TxInput{txinput},
		// // 	TxOutput:       []types.TxOutput{txoutput},
		// // 	CoinbaseProofs: nil,
		// // }
		// // tx.Txid = tx.Hash()

		// // request := &pb.CreateLockTransactionRequest{
		// // 	Tx: tx.ToProto(),
		// // }

		// // response, err := client.CreateLockTransaction(context.Background(), request)
		// // if err != nil {
		// // 	fmt.Println(err)
		// // 	return
		// // }
		// // fmt.Println(response.IsSuccess)
		// // fmt.Println(hexutil.Encode(tx.Txid[:]))
		// // break

		// // amount2 := int64(32768)
		// // txidstring := "0xf91f93e529d2c8f3d5b16b7df68d6a4a9a4485a2cf70cc8302866942c73f3a9f"
		// // txids, err := hexutil.Decode(txidstring)
		// // if err != nil {
		// // 	fmt.Println(err)
		// // 	return

		// // }
		// // txinput2 := types.TxInput{
		// // 	Txid:      crypto.Convert(txids),
		// // 	Voutput:   0,
		// // 	Scriptsig: addr,
		// // 	Value:     amount2,
		// // 	BciType:   0,
		// // 	Address:   addr,
		// // }

		// // str2 := "0xd9689cd5142aeb4684c64ec4700c7ec3b4d18fd4c939e42078faad8e1e4ab41cbf5618e2a4d94189699c1c89ce998dce575bc9793c44b0074a24a98db3a42f3c8ca5e1553e0729518976289d5119705a5d85d77b968e39309d91d5c899bdba510b186ec265cf49303a258d68b3cbb44c5984bba1877a7cb26b803c1a10408b09e33a059813cd8c322552da9e96486d7a875abf0e4eba2406639c1371a002d12f4f5e8652c345804bc08fda1d8c967f175414950bd478cb4f1b4ebb3f850812ad7c0ce9396cdcdd77d323a58fbcb0947cd9431f8e843a440c9a6245e92b555c242392beb88f518bb49bcc5e7c812a3f043b2c9ed71225d3876f4206475b7d2f8ebb21241a02e924d00ac66eb9d1100349096b33c8d61bdbb852a58dbd73e3c1c2bf868335d267a3a12a26981f980d71bd12456796a9470d88c42053d25a2c7f61a5c882ca4d3a406852e2726cde36956530923d6739647233a28815041d450d573d1f46562d638f546986d97788b36109029de770082c52439d66db979c71dec390776b2b4c215d93874b68bf180ca6e0424c6c92888022a52d1358b8ce0a5dde5221a7c7e9940c854e6c2a4a1f2511b3a35a51e5b67a20101e5d79cca72851260f48c6b71d4f5d2124a2e88e00c95b86096d4f92030d50b3d562c06bda8d1149755a3f15c6bfc5675875eba87a78776706580441b2cc99e7cbad7f9bd7439c33562e93ec11e6b4da70dfbabad5b6ea934016110a38a6ae8b90789fd1d857c22a5d6751ecc92d66b1766f4f21afc85cde5bdf77666a65cf207328e0139b12a5e7bbc4049d4fc75d2418ec394ae3736f2483893591021b58e56268b171d8918b2e52184be3df17b0af518f2353cc9620e1db33249eb899af99961eabd555504bdc8ccd5296e449545c35dcac235507254f44693b400c5223e1cfa3ec892b9b581965220a170da0cbec58389847375be8009f760f75a22f0131615e7393ec3175500b227798b0916f2681d60682041c264b9393df8950c035bdea60355626374d1d6ea322e34d9c151c275d25bee3c4011fd33c4192616e8bc69de47f5fc9ba6dbaec3baf18a3dc82821e0a1621b60d9aaf3b3e8b58711a074aac87de99268ab62dca1ea0855a6dd4647b68a5b44cc8dad065d7d898e1bb172b4a28e5078be28d21b43856922cc9a7676f819a2217a46c1ba80ce72036dee40fcfb9a4a0ab7d0db3474fa0d539ddbaa26a531298870fdbbfac2743635ae980526e19ec6429b3c0314571b4cb4f41a97d33ebc3aeb5210a38ca3a75047281a3ce8721380ebe3727a1787046979a679dc6a9681f8e63d0e989aac36cd353a896a47a6342532baf50b55e91410c87c59b3a67116555b413c0e52aa9925a326a9ea704d8eaafca7bad83448767604fde207650664d56216abbebe67182ceb4dc8b0484540d98e340976787ccd6d1a94452c82a3874d0c980ce1be046939fdc0aa803cd715bde625be238cc04d3b73a3922cdc3bedf739829d5165d9b9dbb055f02461c314765a14da1776c9bd4d2370cc6b11b97c022974398e4077972a40c5d45194264d84684bc6c5c9f5906816b0a93eceb4d9fc97b1a9f991e51eabba92f43741e685579c43a1521d73662d29f2eea74bc8e077173d19e9acbe0cc63410004a82eb11a3e80e563725b005a3e82c6ebc0df5d310d46c5a13fb4062602d1a278e7e67ce4d233e6cb640e8a903acaaed7b9377b8d2f4e3b14071e8771afe7eb1cac6078bd2e1daed5d1659a8ed2a1ca5ddb2f9a9a25c03be7511939144c3d928aadd5da17918d3cbe4713c08f573665704cb49d4870e25e6f4b5ddb82eab7893a2658a4911a510c25403ae5d78e3a039a8e55bda95395a7df069fb64728c761ca449634320185722c86829ccbc46c85621022b816e520906fced535d17d4d0d03cb1e95a91720b21aa38e5f0962e15a154aa4964323e09529df1de54a7dcb7dc52c5a159739587cc76b614d12b73ccfaedae64d8b8835980915c188307de3a9b756b6b93869186b62344553ada229b82cc84aca9495442fb4e9e7a0e4026415e3ae6ee1302fd8dfd9535b4c70423f5b723e28a1dae439e404b0857aab1eb70536a26c66d48597101157c6cec407648fdf80a31e1fa94cd5da14d9c9ab9b2c18a43be94e1d1dab07e075df58e3867517c134b55e757fb380dec83c9093ec952b8b7958daeb38bd687230"
		// // addr2, err := hexutil.Decode(str2)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return

		// }

		ethclient, err := ethclient.Dial("http://111.119.239.159:36054")
		if err != nil {
			fmt.Println(err)
			return

		}
		testkey, err := ethcrypto.GenerateKey()
		if err != nil {
			fmt.Println(err)
			return
		}
		readfill, _ := os.OpenFile("key", os.O_RDWR, 0644)
		keydata := make([]byte, testkey.Params().BitSize/8)
		_, err = readfill.Read(keydata)

		if err != nil {
			fmt.Println(err)
			return
		}
		key, err := ethcrypto.ToECDSA(keydata)
		if err != nil {
			fmt.Println(err)
			return
		}
		nonce, err := ethclient.PendingNonceAt(context.Background(), ethcrypto.PubkeyToAddress(key.PublicKey))
		if err != nil {
			fmt.Println(err)
			return
		}

		gaslimit := uint64(32000)
		gasprice, err := ethclient.SuggestGasPrice(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}
		toaddress := ethcrypto.PubkeyToAddress(key.PublicKey)

		chainID, err := ethclient.ChainID(context.Background())
		if err != nil {
			fmt.Println(err)
			return
		}

		amountbyte := big.NewInt(amount)
		data := append([]byte{0x0D, 0x02}, amountbyte.Bytes()...)
		txs := ethtype.NewTransaction(nonce, toaddress, big.NewInt(amount), gaslimit, gasprice, data)
		signedtx, err := ethtype.SignTx(txs, ethtype.NewEIP155Signer(chainID), key)
		if err != nil {
			fmt.Println(err)
			return
		}
		txdata, err := signedtx.MarshalBinary()
		if err != nil {
			fmt.Println(err)
			return
		}

		txoutput2 := types.TxOutput{
			Address:   pot.BurnoutAddress,
			Value:     amount,
			BurnLock:  0,
			ScriptPk:  pot.BurnoutAddress,
			LockTime:  7,
			Rate:      pot.TenYearRate,
			CreatedAt: 11,
			Data:      txdata,
		}
		tx2 := types.RawTx{
			Txid:           [32]byte{},
			TransactionFee: 1,
			TxInput:        []types.TxInput{txinput},
			TxOutput:       []types.TxOutput{txoutput2},
			CoinbaseProofs: nil,
		}
		tx2.Txid = tx2.Hash()
		fmt.Println(hexutil.Encode(tx2.Txid[:]))

		request2 := &pb.CreateDevastateTransactionRequest{
			Tx: tx2.ToProto(),
		}

		response2, err := client.CreateDevastateTransaction(context.Background(), request2)

		if err != nil {
			fmt.Println(err)
			return

		}
		fmt.Println(response2.IsSuccess)
		return
		return
	}

	//privkey := crypto.GenerateKey()
	////pubkey := privkey.PublicKey()
	//fmt.Println(hexutil.Encode(privkey.PublicKeyBytes()))CommitBlock
}
