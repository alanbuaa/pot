package node

import (
	"github.com/sirupsen/logrus"
	"github.com/zzz136454872/upgradeable-consensus/config"
)

// registerNodeDiscoveryCallback 注册节点发现回调
func (node *Node) registerNodeDiscoveryCallback() {
	node.p2pAdaptor.RegisterNodeDiscoveryCallback(node)
	node.log.Info("Node discovery callback registered")
}

// OnNodeDiscovered 实现 NodeDiscoveryCallback 接口 - 当发现新节点时调用
func (node *Node) OnNodeDiscovered(nodeID int64, address string) error {
	node.log.WithFields(logrus.Fields{
		"node_id": nodeID,
		"address": address,
	}).Info("Node discovered")

	// 如果节点信息不在 Nodes 中，添加它
	if _, err := node.config.GetNodeFromSet(nodeID); err != nil {
		// 创建新节点信息
		newNode := &config.NodeInfo{
			ID:         nodeID,
			P2PAddress: address,
		}
		if err := node.config.AddNode(newNode); err != nil {
			node.log.WithError(err).Warn("Failed to add discovered node to Nodes")
			return err
		}
		node.log.WithField("node_id", nodeID).Info("Added discovered node to Nodes")
	}
	return nil
}

// OnNodeLost 实现 NodeDiscoveryCallback 接口 - 当节点丢失时调用
func (node *Node) OnNodeLost(nodeID int64) error {
	node.log.WithField("node_id", nodeID).Warn("Node lost")
	// 可以选择是否从 Nodes 中移除节点
	// 对于非动态共识，通常保留节点信息
	return nil
}
