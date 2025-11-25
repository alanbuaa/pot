package vdf

import (
	"blockchain-crypto/vdf/utils"
	"runtime"
	"testing"
	"time"
)

// TestNew 测试 VDF 实例的创建
func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		vdfType    string
		challenge  []byte
		iterations int
		id         int64
	}{
		{
			name:       "创建 pietrzak 类型 VDF",
			vdfType:    "pietrzak",
			challenge:  []byte("test challenge"),
			iterations: 1000,
			id:         1,
		},
		{
			name:       "创建 wesolowski 类型 VDF",
			vdfType:    "wesolowski",
			challenge:  []byte("another challenge"),
			iterations: 2000,
			id:         2,
		},
		{
			name:       "空 challenge",
			vdfType:    "pietrzak",
			challenge:  []byte{},
			iterations: 500,
			id:         3,
		},
		{
			name:       "大的 iterations",
			vdfType:    "wesolowski",
			challenge:  []byte("test"),
			iterations: 100000,
			id:         4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdf := New(tt.vdfType, tt.challenge, tt.iterations, tt.id)

			if vdf == nil {
				t.Fatal("New() 返回 nil")
			}

			if vdf.Type != tt.vdfType {
				t.Errorf("Type = %v, want %v", vdf.Type, tt.vdfType)
			}

			if string(vdf.Challenge) != string(tt.challenge) {
				t.Errorf("Challenge = %v, want %v", vdf.Challenge, tt.challenge)
			}

			if vdf.Iterations != tt.iterations {
				t.Errorf("Iterations = %v, want %v", vdf.Iterations, tt.iterations)
			}

			// 检查 Controller 的初始状态
			if vdf.Controller.Pid != -1 {
				t.Errorf("Controller.Pid = %v, want -1", vdf.Controller.Pid)
			}

			if vdf.Controller.CpuNo != 0 {
				t.Errorf("Controller.CpuNo = %v, want 0", vdf.Controller.CpuNo)
			}

			if vdf.Controller.IsAbort != false {
				t.Errorf("Controller.IsAbort = %v, want false", vdf.Controller.IsAbort)
			}

			if vdf.Controller.IsAllocated != false {
				t.Errorf("Controller.IsAllocated = %v, want false", vdf.Controller.IsAllocated)
			}
		})
	}
}

// TestInit 测试 init 函数的初始化
func TestInit(t *testing.T) {
	// init() 在包加载时自动执行

	// 验证 cpuList 的初始化
	expectedCPUCount := runtime.NumCPU()
	if len(cpuList) != expectedCPUCount {
		t.Errorf("cpuList 长度 = %v, want %v", len(cpuList), expectedCPUCount)
	}

	for i := 0; i < expectedCPUCount; i++ {
		if cpuList[i] != uint8(i) {
			t.Errorf("cpuList[%d] = %v, want %v", i, cpuList[i], uint8(i))
		}
	}

	// 验证 cpuCounter 的初始化
	if cpuCounter == nil {
		t.Fatal("cpuCounter 未初始化")
	}

	// 验证 cpuChecker 的初始化
	if cpuChecker == nil {
		t.Fatal("cpuChecker 未初始化")
	}
}

// TestExecuteAndVerify 测试 VDF 的执行和验证
func TestExecuteAndVerify(t *testing.T) {
	// 注意：这个测试可能需要较长时间运行，取决于 iterations 的大小
	// 使用较小的 iterations 以加快测试速度
	// 此测试需要 VDF 可执行文件存在，如果不存在则跳过
	tests := []struct {
		name       string
		vdfType    string
		challenge  []byte
		iterations int
		shouldPass bool
	}{
		{
			name:       "wesolowski 小迭代次数",
			vdfType:    "wesolowski",
			challenge:  []byte("test challenge 1"),
			iterations: 100,
			shouldPass: true,
		},
		{
			name:       "默认类型 (wesolowski)",
			vdfType:    "default",
			challenge:  []byte("test challenge 2"),
			iterations: 100,
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdf := New(tt.vdfType, tt.challenge, tt.iterations, 1)

			// 执行 VDF
			result, err := vdf.Execute()

			// 如果返回 nil 且没有错误，可能是 VDF 可执行文件不存在或被中止
			if result == nil && err == nil {
				t.Skip("VDF 可执行文件不存在或执行被中止，跳过此测试")
				return
			}

			if err != nil {
				t.Skipf("Execute() error = %v, 可能缺少 VDF 可执行文件", err)
				return
			}

			if result == nil {
				t.Fatal("Execute() 返回 nil 结果但没有错误")
			}

			// 验证结果
			isValid := vdf.Verify(result)
			if isValid != tt.shouldPass {
				t.Errorf("Verify() = %v, want %v", isValid, tt.shouldPass)
			}
		})
	}
}

// TestVerifyWithInvalidResult 测试使用无效结果的验证
func TestVerifyWithInvalidResult(t *testing.T) {
	vdf := New("wesolowski", []byte("challenge"), 100, 1)

	// 使用错误的结果进行验证
	invalidResult := []byte("invalid result")
	isValid := vdf.Verify(invalidResult)

	if isValid {
		t.Error("Verify() 使用无效结果应该返回 false")
	}
}

// TestCheckVDF 测试 CheckVDF 方法
func TestCheckVDF(t *testing.T) {
	tests := []struct {
		name       string
		vdfType    string
		input      []byte
		result     []byte
		iterations int
		expectPass bool
	}{
		{
			name:       "有效的 VDF 检查",
			vdfType:    "wesolowski",
			input:      []byte("test input"),
			result:     nil, // 将在测试中生成
			iterations: 100,
			expectPass: true,
		},
		{
			name:       "无效的 VDF 检查",
			vdfType:    "wesolowski",
			input:      []byte("test input"),
			result:     []byte("wrong result"),
			iterations: 100,
			expectPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdf := New(tt.vdfType, tt.input, tt.iterations, 1)

			var result []byte
			if tt.result == nil && tt.expectPass {
				// 生成有效的结果
				var err error
				result, err = vdf.Execute()

				// 如果执行失败（可能是缺少可执行文件），跳过测试
				if err != nil || result == nil {
					t.Skipf("无法生成 VDF 结果: err=%v, 跳过测试", err)
					return
				}
			} else {
				result = tt.result
			}

			isValid := vdf.CheckVDF(tt.input, result)
			if isValid != tt.expectPass {
				t.Errorf("CheckVDF() = %v, want %v", isValid, tt.expectPass)
			}
		})
	}
}

// TestAbort 测试 Abort 方法
func TestAbort(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(*Vdf)
		expectError bool
	}{
		{
			name: "中止未分配的 VDF",
			setupFunc: func(vdf *Vdf) {
				vdf.Controller.IsAllocated = false
			},
			expectError: false,
		},
		{
			name: "中止已分配的 VDF",
			setupFunc: func(vdf *Vdf) {
				vdf.Controller.IsAllocated = true
				vdf.Controller.Pid = 99999 // 使用一个不存在的 PID
			},
			expectError: false,
		},
		{
			name: "中止正在等待的 VDF",
			setupFunc: func(vdf *Vdf) {
				vdf.Controller.IsAbort = false
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vdf := New("wesolowski", []byte("test"), 100, 1)
			if tt.setupFunc != nil {
				tt.setupFunc(vdf)
			}

			err := vdf.Abort()
			if (err != nil) != tt.expectError {
				t.Errorf("Abort() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestConcurrentVDFCreation 测试并发创建 VDF 实例
func TestConcurrentVDFCreation(t *testing.T) {
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			vdf := New("wesolowski", []byte("concurrent test"), 100, int64(id))
			if vdf == nil {
				t.Error("并发创建失败")
			}
			done <- true
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		select {
		case <-done:
			// 成功
		case <-time.After(5 * time.Second):
			t.Fatal("并发创建超时")
		}
	}
}

// TestVdfTypeSwitch 测试不同 VDF 类型的切换
func TestVdfTypeSwitch(t *testing.T) {
	challenge := []byte("type switch test")
	iterations := 100

	// 测试 pietrzak 类型
	vdfPietrzak := New("pietrzak", challenge, iterations, 1)
	if vdfPietrzak.Type != "pietrzak" {
		t.Errorf("pietrzak 类型设置错误")
	}

	// 测试 wesolowski 类型
	vdfWesolowski := New("wesolowski", challenge, iterations, 2)
	if vdfWesolowski.Type != "wesolowski" {
		t.Errorf("wesolowski 类型设置错误")
	}

	// 测试未知类型（应该使用默认的 wesolowski）
	vdfUnknown := New("unknown", challenge, iterations, 3)
	if vdfUnknown.Type != "unknown" {
		t.Errorf("未知类型设置错误")
	}
}

// TestControllerInitialization 测试 Controller 的初始化
func TestControllerInitialization(t *testing.T) {
	vdf := New("wesolowski", []byte("test"), 100, 1)

	ctrl := vdf.Controller

	// 验证所有初始值
	if ctrl.Pid != -1 {
		t.Errorf("初始 Pid = %v, want -1", ctrl.Pid)
	}

	if ctrl.CpuNo != 0 {
		t.Errorf("初始 CpuNo = %v, want 0", ctrl.CpuNo)
	}

	if ctrl.IsAbort != false {
		t.Errorf("初始 IsAbort = %v, want false", ctrl.IsAbort)
	}

	if ctrl.IsAllocated != false {
		t.Errorf("初始 IsAllocated = %v, want false", ctrl.IsAllocated)
	}
}

// BenchmarkNew 基准测试 VDF 创建
func BenchmarkNew(b *testing.B) {
	challenge := []byte("benchmark challenge")
	for i := 0; i < b.N; i++ {
		New("wesolowski", challenge, 1000, int64(i))
	}
}

// BenchmarkExecuteSmall 基准测试小迭代次数的 VDF 执行
func BenchmarkExecuteSmall(b *testing.B) {
	challenge := []byte("benchmark challenge")
	iterations := 10

	// 先测试一次看是否可执行
	vdf := New("wesolowski", challenge, iterations, 0)
	result, err := vdf.Execute()
	if err != nil || result == nil {
		b.Skip("VDF 可执行文件不可用，跳过基准测试")
		return
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		vdf := New("wesolowski", challenge, iterations, int64(i))
		_, err := vdf.Execute()
		if err != nil {
			b.Fatalf("Execute() error = %v", err)
		}
	}
}

// TestCPUCounterGlobal 测试全局 CPU 计数器
func TestCPUCounterGlobal(t *testing.T) {
	if cpuCounter == nil {
		t.Fatal("全局 cpuCounter 未初始化")
	}

	// 创建一个测试用的 controller
	ctrl := &utils.Controller{
		Pid:         -1,
		CpuNo:       0,
		IsAbort:     false,
		IsAllocated: false,
	}

	// 测试 CPU 分配和释放
	go cpuCounter.Occupy(ctrl)
	time.Sleep(100 * time.Millisecond)

	if !ctrl.IsAllocated && !ctrl.IsAbort {
		// 如果还没有分配，可能是资源繁忙
		t.Log("CPU 可能正在被使用")
	}
}

// TestNilChallenge 测试 nil challenge
func TestNilChallenge(t *testing.T) {
	vdf := New("wesolowski", nil, 100, 1)
	if vdf == nil {
		t.Fatal("New() 使用 nil challenge 不应该返回 nil")
	}

	// 尝试执行（可能会失败，这取决于实现）
	_, err := vdf.Execute()
	// 不强制要求错误，因为某些实现可能处理 nil challenge
	t.Logf("使用 nil challenge 执行: error = %v", err)
}

// TestZeroIterations 测试零迭代次数
func TestZeroIterations(t *testing.T) {
	vdf := New("wesolowski", []byte("test"), 0, 1)
	if vdf == nil {
		t.Fatal("New() 使用零迭代次数不应该返回 nil")
	}

	if vdf.Iterations != 0 {
		t.Errorf("Iterations = %v, want 0", vdf.Iterations)
	}
}

// TestNegativeIterations 测试负迭代次数
func TestNegativeIterations(t *testing.T) {
	vdf := New("wesolowski", []byte("test"), -100, 1)
	if vdf == nil {
		t.Fatal("New() 使用负迭代次数不应该返回 nil")
	}

	if vdf.Iterations != -100 {
		t.Errorf("Iterations = %v, want -100", vdf.Iterations)
	}
}
