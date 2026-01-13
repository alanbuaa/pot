package upgrade

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MessageCache 消息缓存
// 用于在共识切换期间缓存消息，处理非同步网络场景
// 场景1: 切换前到达的属于新共识的消息
// 场景2: 切换后到达的属于旧共识的消息
type MessageCache struct {
	// 新共识消息缓存（切换前收到的）
	newConsensusMessages map[string]*CachedMessage

	// 旧共识消息缓存（切换后收到的）
	oldConsensusMessages map[string]*CachedMessage

	// 缓存配置
	maxCacheSize   int
	messageTimeout time.Duration

	// 切换状态
	switched       bool
	switchHeight   uint64
	newConsensusID int64
	oldConsensusID int64

	log *logrus.Entry
	mu  sync.RWMutex
}

// CachedMessage 缓存的消息
type CachedMessage struct {
	Message              interface{} // 通用消息类型，可以是任何共识消息
	ConsensusID          int64       // 发送者共识ID
	ReceiverConsensusID  int64       // 接收者共识ID（关键字段）
	Epoch                uint64      // 消息所属epoch/时段（关键字段）
	ReceivedTime         time.Time
	ProcessedAfterSwitch bool
}

// NewMessageCache 创建消息缓存
func NewMessageCache(log *logrus.Entry) *MessageCache {
	if log == nil {
		log = logrus.NewEntry(logrus.New())
	}

	return &MessageCache{
		newConsensusMessages: make(map[string]*CachedMessage),
		oldConsensusMessages: make(map[string]*CachedMessage),
		maxCacheSize:         10000,
		messageTimeout:       5 * time.Minute,
		log:                  log,
	}
}

// SetSwitchInfo 设置切换信息
func (mc *MessageCache) SetSwitchInfo(switchHeight uint64, oldConsensusID, newConsensusID int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.switchHeight = switchHeight
	mc.oldConsensusID = oldConsensusID
	mc.newConsensusID = newConsensusID

	mc.log.WithFields(logrus.Fields{
		"switch_height":    switchHeight,
		"old_consensus_id": oldConsensusID,
		"new_consensus_id": newConsensusID,
	}).Info("Message cache switch info set")
}

// CacheMessage 缓存消息
// receiverConsensusID: 接收者共识ID
// epoch: 消息所属epoch
func (mc *MessageCache) CacheMessage(msg interface{}, senderConsensusID, receiverConsensusID int64, epoch uint64) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// 生成消息键（包含epoch信息）
	key := fmt.Sprintf("%d_%d_%d", receiverConsensusID, epoch, time.Now().UnixNano())

	cached := &CachedMessage{
		Message:             msg,
		ConsensusID:         senderConsensusID,
		ReceiverConsensusID: receiverConsensusID,
		Epoch:               epoch,
		ReceivedTime:        time.Now(),
	}

	// 根据切换状态决定缓存位置
	if !mc.switched {
		// 切换前：如果是新共识的消息，缓存起来
		if receiverConsensusID == mc.newConsensusID {
			if len(mc.newConsensusMessages) >= mc.maxCacheSize {
				return fmt.Errorf("new consensus message cache full")
			}
			mc.newConsensusMessages[key] = cached
			mc.log.WithFields(logrus.Fields{
				"key":                key,
				"receiver_consensus": receiverConsensusID,
				"epoch":              epoch,
			}).Debug("Cached new consensus message (pre-switch)")
		}
	} else {
		// 切换后：如果是旧共识的消息，缓存起来（可能是延迟到达的）
		if receiverConsensusID == mc.oldConsensusID {
			if len(mc.oldConsensusMessages) >= mc.maxCacheSize {
				return fmt.Errorf("old consensus message cache full")
			}
			mc.oldConsensusMessages[key] = cached
			mc.log.WithFields(logrus.Fields{
				"key":                key,
				"receiver_consensus": receiverConsensusID,
				"epoch":              epoch,
			}).Debug("Cached old consensus message (post-switch)")
		}
	}

	return nil
}

// OnSwitch 切换时调用
// 将缓存的新共识消息移交给新共识处理
func (mc *MessageCache) OnSwitch() []*CachedMessage {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.switched = true

	// 返回所有缓存的新共识消息
	messages := make([]*CachedMessage, 0, len(mc.newConsensusMessages))
	for _, msg := range mc.newConsensusMessages {
		msg.ProcessedAfterSwitch = true
		messages = append(messages, msg)
	}

	mc.log.WithField("count", len(messages)).Info("Delivering cached new consensus messages after switch")

	// 清空新共识消息缓存
	mc.newConsensusMessages = make(map[string]*CachedMessage)

	return messages
}

// PopAllForConsensus 获取并移除特定共识的所有缓存消息（用于候选共识启动时）
func (mc *MessageCache) PopAllForConsensus(consensusID int64) []*CachedMessage {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var messages []*CachedMessage

	if !mc.switched {
		// 切换前：从新共识缓存中取出
		if consensusID == mc.newConsensusID {
			for key, msg := range mc.newConsensusMessages {
				if msg.ReceiverConsensusID == consensusID {
					messages = append(messages, msg)
					delete(mc.newConsensusMessages, key)
				}
			}
		}
	} else {
		// 切换后：从旧共识缓存中取出
		if consensusID == mc.oldConsensusID {
			for key, msg := range mc.oldConsensusMessages {
				if msg.ReceiverConsensusID == consensusID {
					messages = append(messages, msg)
					delete(mc.oldConsensusMessages, key)
				}
			}
		}
	}

	mc.log.WithFields(logrus.Fields{
		"consensus_id": consensusID,
		"count":        len(messages),
	}).Info("Popped cached messages for consensus")

	return messages
}

// GetCachedMessages 获取缓存的消息
func (mc *MessageCache) GetCachedMessages(consensusID int64) []*CachedMessage {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var messages []*CachedMessage

	if consensusID == mc.newConsensusID && !mc.switched {
		// 切换前获取新共识消息
		for _, msg := range mc.newConsensusMessages {
			messages = append(messages, msg)
		}
	} else if consensusID == mc.oldConsensusID && mc.switched {
		// 切换后获取旧共识消息
		for _, msg := range mc.oldConsensusMessages {
			messages = append(messages, msg)
		}
	}

	return messages
}

// CleanupExpiredMessages 清理过期消息
func (mc *MessageCache) CleanupExpiredMessages() int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	now := time.Now()
	cleaned := 0

	// 清理新共识消息缓存
	for key, msg := range mc.newConsensusMessages {
		if now.Sub(msg.ReceivedTime) > mc.messageTimeout {
			delete(mc.newConsensusMessages, key)
			cleaned++
		}
	}

	// 清理旧共识消息缓存
	for key, msg := range mc.oldConsensusMessages {
		if now.Sub(msg.ReceivedTime) > mc.messageTimeout {
			delete(mc.oldConsensusMessages, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		mc.log.WithField("count", cleaned).Info("Cleaned up expired messages")
	}

	return cleaned
}

// Clear 清空所有缓存
func (mc *MessageCache) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.newConsensusMessages = make(map[string]*CachedMessage)
	mc.oldConsensusMessages = make(map[string]*CachedMessage)
	mc.switched = false

	mc.log.Info("Message cache cleared")
}

// GetCacheStats 获取缓存统计
func (mc *MessageCache) GetCacheStats() *CacheStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &CacheStats{
		NewConsensusMessageCount: len(mc.newConsensusMessages),
		OldConsensusMessageCount: len(mc.oldConsensusMessages),
		MaxCacheSize:             mc.maxCacheSize,
		Switched:                 mc.switched,
		SwitchHeight:             mc.switchHeight,
	}
}

// CacheStats 缓存统计
type CacheStats struct {
	NewConsensusMessageCount int
	OldConsensusMessageCount int
	MaxCacheSize             int
	Switched                 bool
	SwitchHeight             uint64
}

// DropByEpoch 清理指定epoch的消息
func (mc *MessageCache) DropByEpoch(epoch uint64) int {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cleaned := 0

	// 清理新共识消息缓存
	for key, msg := range mc.newConsensusMessages {
		if msg.Epoch == epoch {
			delete(mc.newConsensusMessages, key)
			cleaned++
		}
	}

	// 清理旧共识消息缓存
	for key, msg := range mc.oldConsensusMessages {
		if msg.Epoch == epoch {
			delete(mc.oldConsensusMessages, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		mc.log.WithFields(logrus.Fields{
			"epoch": epoch,
			"count": cleaned,
		}).Info("Dropped messages by epoch")
	}

	return cleaned
}

// GetMessagesByEpoch 获取指定epoch的所有消息
func (mc *MessageCache) GetMessagesByEpoch(epoch uint64) []*CachedMessage {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var messages []*CachedMessage

	// 从新共识缓存中查找
	for _, msg := range mc.newConsensusMessages {
		if msg.Epoch == epoch {
			messages = append(messages, msg)
		}
	}

	// 从旧共识缓存中查找
	for _, msg := range mc.oldConsensusMessages {
		if msg.Epoch == epoch {
			messages = append(messages, msg)
		}
	}

	return messages
}

// generateMessageKey 生成消息键
func (mc *MessageCache) generateMessageKey(msg interface{}) string {
	// 使用消息的关键字段生成唯一键
	// 这里简化处理，使用当前时间戳和缓存计数
	if msg == nil {
		return ""
	}
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), len(mc.newConsensusMessages)+len(mc.oldConsensusMessages))
}

// SetMaxCacheSize 设置最大缓存大小
func (mc *MessageCache) SetMaxCacheSize(size int) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.maxCacheSize = size
}

// SetMessageTimeout 设置消息超时时间
func (mc *MessageCache) SetMessageTimeout(timeout time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.messageTimeout = timeout
}

// IsSwitched 检查是否已切换
func (mc *MessageCache) IsSwitched() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.switched
}

// StartCleanupTask 启动清理任务
func (mc *MessageCache) StartCleanupTask(interval time.Duration) chan struct{} {
	stopChan := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				mc.CleanupExpiredMessages()
			case <-stopChan:
				mc.log.Info("Message cache cleanup task stopped")
				return
			}
		}
	}()

	mc.log.WithField("interval", interval).Info("Message cache cleanup task started")
	return stopChan
}
