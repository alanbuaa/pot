package pot

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/zzz136454872/upgradeable-consensus/crypto"
	"github.com/zzz136454872/upgradeable-consensus/pb"
	"github.com/zzz136454872/upgradeable-consensus/types"
	"golang.org/x/exp/rand"
)

func TestTestTx(t *testing.T) {
	// TestTestTx: 测试构造一个简单的交易、计算其 Txid 并打印相关字段。
	// 目的：验证 Tx 的序列化/哈希和字段访问没有错误。
	txid, _ := hexutil.Decode("0x89e5feab71c9ccd7f3a999605d012eaf208703a6130b91dc6328fd4a9e976d0d")
	addr, _ := hexutil.Decode("0xa47155b42648816998e576ffbfa045ab00000000000000007662a30d4b5875b73a8768fcf01bf52d2326a7660e24033a")
	input := types.TxInput{
		Txid:      crypto.Convert(txid),
		Voutput:   0,
		Scriptsig: addr,
		Address:   addr,
		Value:     32768,
		BciType:   1,
	}
	data, _ := hexutil.Decode("0x00")
	output := types.TxOutput{
		Address:  addr,
		Value:    32768,
		BciType:  1,
		Interest: 10,
		LockTime: 7,
		Data:     data,
		BurnLock: 62560,
		Rate:     0.00,
	}

	tx := types.RawTx{
		TxInput:        []types.TxInput{input},
		TxOutput:       []types.TxOutput{output},
		TransactionFee: 0,
	}
	tx.Txid = tx.Hash()
	fmt.Println(hexutil.Encode(tx.Txid[:]))
	fmt.Println(hexutil.Encode(data))
	fmt.Println(tx.TxOutput[0].Rate)
	t.Log(hexutil.Encode(tx.Txid[:]))
}

func TestBci(t *testing.T) {
	// TestBci: 测试 BCI 奖励的分组、权重计算与概率选择逻辑。
	// 目的：使用固定的 vdf/res 随机种子验证按权重抽样与分组正常工作。
	vdf1res := []byte("abcdefg789456121323")
	rand.Seed(binary.BigEndian.Uint64(vdf1res[:8]))
	//test := big.NewInt(0)
	Bcireward1 := &BciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  10,
		Proof:   BciProof{},
		BciType: 1,
		weight:  0,
	}
	Bcireward2 := &BciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  5,
		Proof:   BciProof{},
		BciType: 1,
		weight:  0,
	}
	Bcireward3 := &BciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  1,
		Proof:   BciProof{},
		BciType: 1,
		weight:  0,
	}
	Bcireward4 := &BciReward{
		Address: big.NewInt(rand.Int63()).Bytes(),
		Amount:  5,
		Proof:   BciProof{},
		BciType: 1,
		weight:  0,
	}

	Bcirewards := []*BciReward{Bcireward1, Bcireward2, Bcireward3, Bcireward4}
	groupsdata := groupByChainID(Bcirewards)
	for _, rewards := range groupsdata {
		total := int64(0)
		for _, reward := range rewards {
			total += reward.Amount
		}
		for _, reward := range rewards {
			reward.weight = float64(reward.Amount) / float64(total)
			//t.Log(reward.weight)
		}
	}
	selectreward := make(map[int32][]*BciReward)
	vdf0res := []byte("abcdefg789456121323sssswererewrerwwerssssessssssss")
	for chainID, rewards := range groupsdata {
		t.Log(rewards)
		sort.Slice(rewards, func(i, j int) bool {
			return bytes.Compare(rewards[i].Address, rewards[j].Address) < 0
		})
		t.Log(rewards)
		for _, reward := range rewards {
			t.Log(reward.weight)
		}
		rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
		for i := 0; i < Selectn; i++ {
			r := rand.Float64()
			t.Log(r)
			acnum := 0.0

			for _, reward := range rewards {
				//t.Log(reward.weight)
				acnum += reward.weight
				if r <= acnum {
					selectreward[chainID] = append(selectreward[chainID], reward)
					break
				}
			}
		}
	}
	for chainID, rewards := range selectreward {
		t.Log(chainID)
		t.Log(rewards)

	}

}

func TestRandom(t *testing.T) {
	// TestRandom: 演示基于 vdf 哈希种子重复设置 rand.Seed 后的随机数生成一致性。
	vdf0res := []byte("abcdefg789456121323")
	for i := 0; i < 10; i++ {
		rand.Seed(binary.BigEndian.Uint64(crypto.Hash(vdf0res)[:8]))
		t.Log(rand.Float64())
		t.Log(rand.Float64())
	}
}

//	func TestBoltDbView(t *testing.T) {
//		// 创建一个临时的 BoltDB 数据库文件
//		db, err := bolt.Open("test.db", 0600, nil)
//		if err != nil {
//			t.Fatal(err)
//		}
//		defer db.Close()
//
//		// 写入一些测试数据
//		err = db.Update(func(tx *bolt.Tx) error {
//			b, err := tx.CreateBucketIfNotExists([]byte(storage.UTXOBucket))
//			if err != nil {
//				return err
//			}
//
//			// 假设这里有一些测试数据
//			testData := []byte("some-test-data")
//			err = b.Put([]byte("key"), testData)
//			if err != nil {
//				return err
//			}
//			return nil
//		})
//		if err != nil {
//			t.Fatal(err)
//		}
//
//		// 测试 View 函数
//		err = db.View(func(tx *bolt.Tx) error {
//			b := tx.Bucket([]byte(storage.UTXOBucket))
//			if b == nil {
//				t.Errorf("Bucket not found")
//				return fmt.Errorf("bucket not found")
//			}
//
//			c := b.Cursor()
//			count := 0
//			for k, v := c.First(); k != nil; k, v = c.Next() {
//				outs := types.DecodeByte2Outputs(v)
//				if err != nil {
//					t.Errorf("Failed to decode outputs: %v", err)
//					return err
//				}
//
//				for _, out := range outs {
//					if out.IsLockedWithKey(address) {
//						// 假设 utxos 是一个全局变量
//						utxos = append(utxos, out)
//					}
//				}
//				count++
//			}
//			if count != 1 { // 假设只有一个键值对
//				t.Errorf("Expected 1 item, got %d", count)
//			}
//			return nil
//		})
//
//		if err != nil {
//			t.Errorf("Error during bolt db view: %v", err)
//		}
//	}
func TestBytesEqual(t *testing.T) {
	// TestBytesEqual: 验证数组直接比较与 bytes.Equal 在 []byte/[32]byte 场景下的行为。
	first := [32]byte{}
	second := &types.RawTx{
		Txid: [32]byte{},
	}
	t.Log(first == second.Txid)
	t.Log(bytes.Equal(first[:], second.Txid[:]))
}

func TestUTXO(t *testing.T) {

	// TestUTXO: 演示生成 UTXO key 字符串并将其分解为 txid 和 voutput。
	// 目的：验证对 utxokey 格式的构造与解析逻辑没有问题。
	//var testData = []*types.TxOutput{
	//	{
	//		Address:  []byte("address1"),
	//		Value:    1000,
	//		IsCoinbase:  false,
	//		ScriptPk: []byte("script1"),
	//		Proof:    []byte("proof1"),
	//		LockTime: time.Now().Unix(),
	//	},
	//	//{
	//	//	Address:  []byte("address2"),
	//	//	Value:    2000,
	//	//	IsCoinbase:  true,
	//	//	ScriptPk: []byte("script2"),
	//	//	Proof:    []byte("proof2"),
	//	//	LockTime: time.Now().Add(time.Hour * 24).Unix(), // 锁定时间为当前时间加上一天
	//	//},
	//	//{
	//	//	Address:  []byte("address3"),
	//	//	Value:    3000,
	//	//	IsCoinbase:  false,
	//	//	ScriptPk: []byte("script3"),
	//	//	Proof:    []byte("proof3"),
	//	//	LockTime: time.Now().Add(time.Hour * 48).Unix(), // 锁定时间为当前时间加上两天
	//	//},
	//	//{
	//	//	Address:  []byte("address4"),
	//	//	Value:    4000,
	//	//	IsCoinbase:  true,
	//	//	ScriptPk: []byte("script4"),
	//	//	Proof:    []byte("proof4"),
	//	//	LockTime: time.Now().Add(time.Hour * 72).Unix(), // 锁定时间为当前时间加上三天
	//	//},
	//	//{
	//	//	Address:  []byte("address5"),
	//	//	Value:    5000,
	//	//	IsCoinbase:  false,
	//	//	ScriptPk: []byte("script5"),
	//	//	Proof:    []byte("proof5"),
	//	//	LockTime: time.Now().Add(time.Hour * 96).Unix(), // 锁定时间为当前时间加上四天
	//	//},
	//}
	//testData[0] = &types.TxOutput{}
	s := fmt.Sprintf("%s:2", crypto.Convert(types.RandByte()))
	b := []byte(s)
	t.Log(hexutil.Encode(b))
	//t.Log(len(testData))
	t.Log(s)
	t.Log(string(b))

}

func TestBuffer(t *testing.T) {
	// TestBuffer: 演示 bytes.Buffer 的写入、长度和 Bytes() 行为。
	var buffer bytes.Buffer
	buffer.Write([]byte("aaaaaa"))
	buffer.Write([]byte("abcdef"))
	fmt.Println(buffer.String())
	fmt.Println(buffer.Len())
	b := buffer.Bytes()
	fmt.Printf("%v\n", b)
}

func TestBciInterest(t *testing.T) {
	// TestBciInterest: 用于测试和打印利率/时间常量的计算结果。
	year := TenYears + 365*CoinbaseLock
	t.Log(year / TenYears)
}

func TestUtxoKey(t *testing.T) {
	// TestUtxoKey: 生成随机 txid，构造 utxokey 字符串并验证解析、编码/解码流程。

	randbyte := types.RandByte()
	hexStr := hexutil.Encode(randbyte)
	fmt.Println(hexStr)

	// 去除 0x 前缀并去除前后空白字符

	fmt.Println(hexStr)
	utxokey := fmt.Sprintf("%s:%d", hexStr, 0)
	fmt.Println("Generated utxokey:", utxokey)

	// 使用 strings.Split 进行解析
	parts := strings.Split(utxokey, ":")
	if len(parts) != 2 {
		t.Errorf("Invalid utxokey format: %v", utxokey)
		return
	}

	txid := parts[0]
	voutput, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		t.Errorf("Failed to parse voutput: %v", err)
		return
	}

	fmt.Println("Parsed txid:", txid)
	fmt.Println("Parsed voutput:", voutput)

	// 如果需要将 txid 转换回字节切片
	decodedTxid, err := hexutil.Decode(txid)
	if err != nil {
		t.Errorf("Failed to decode txid: %v", err)
		return
	}
	fmt.Println("Decoded txid:", hexutil.Encode(decodedTxid))
	fmt.Println(bytes.Equal(decodedTxid, randbyte))
	id := crypto.Convert(decodedTxid)
	fmt.Println(hexutil.Encode(id[:]))

}

func TestFloatCompare(t *testing.T) {
	// TestFloatCompare: 测试浮点数到 pb 字段的转换及比较，演示精度和 epsilon 容差的使用。
	a := TenYearRate
	b := &pb.TxOutput{
		Rate: float32(a),
	}

	fmt.Println(b.Rate)
	fmt.Println(a == float64(b.Rate))

	c := math.Abs(a-float64(b.Rate)) < epsilon
	fmt.Println(c)
	fmt.Println(epsilon)
}

func TestHash(t *testing.T) {
	// TestHash: 构建一个 RawTx 对象并打印其 JSON 哈希（用于快速检查序列化/哈希逻辑）。
	txinput := types.TxInput{
		Txid:    [32]byte{},
		Voutput: -1,
	}
	addr := "0xa47155b42648816998e576ffbfa045ab00000000000000007662a30d4b5875b73a8768fcf01bf52d2326a7660e24033a"
	add, _ := hexutil.Decode(addr)
	txoutput := types.TxOutput{
		Address:  add,
		BciType:  1,
		BurnLock: 0,
		Data:     []byte("Test"),
		Interest: 0,
		LockTime: 7,
		Proof:    []byte("Test"),
		Rate:     0,
		Value:    32768,
	}
	rawtx := &types.RawTx{
		TxInput:        []types.TxInput{txinput},
		TxOutput:       []types.TxOutput{txoutput},
		TransactionFee: 0,
	}
	b, err := json.Marshal(rawtx)
	if err != nil {
		t.Errorf("Failed to encode rawtx: %v", err)
		return
	}
	fmt.Println(hexutil.Encode(crypto.Hash(b)))
}
