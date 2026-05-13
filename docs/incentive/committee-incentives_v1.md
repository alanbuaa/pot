委员会激励文档
总激励
正向激励（基础部分） 权重δ_1，奖励〖reward〗\_1
正向激励（手续费部分） 权重δ_2，奖励〖reward〗\_2
反向惩罚 权重δ_3，奖励〖reward〗\_3
节点总激励：
totalReward=δ_1*〖reward〗\_1+δ_2*〖reward〗\_2-δ_3\*〖reward〗\_3

    正向激励-基础部分

系统参数：
出块数量的权重 β_1
交易数目的权重 β_2
投票数量的权重 β_3
2.1 Leader节点
变量/工作量：
出块数量 blockNum
交易数目 txNum
Leader收集的肯定票数量 voteNum1
Leader收集的否定票数量 voteNum0
激励：
〖reward〗\_1=β_1*blockNum+β_2*txNum+β_3 (voteNum1+0.5voteNum0)
2.2委员节点
变量/工作量：
被接收的肯定票数量 recVoteNum1
被接收的否定票数量 recVoteNum0
激励：
〖reward〗\_1=β_3 (recVoteNum1+0.5recVoteNum0)

    正向激励-手续费部分

系统参数：
当前委员会权重 w_1
前一个委员会权重 w_2
变量/工作量：
当前委员会总手续费为 c_1
当前委员会规模为 n_1
前一个委员会总手续费为 c_2
前一个委员会规模为 n_2
激励：
〖reward〗\_2=(w_1*c_1)/n_1+(w_2*c_2)/n_2

    反向惩罚

系统参数：
错误的区块权重 δ_1
委员节点对错误区块投肯定票的数量 δ_2
委员节点对正确区块投否定票的数量 δ_3
变量/工作量：
Leader模棱两可的区块数量 errorBlockNum
委员节点对错误区块投肯定票的数量 errorRecVoteNum1
委员节点对正确区块投否定票的数量 errorRecVoteNum0
错误的区块惩罚：
〖negativeReward〗\_1=δ_1*errorBlockNum
错误的投票惩罚：
〖negativeReward〗\_2=δ_2*errorRecVoteNum1+δ_3\*errorRecVoteNum0
激励：
〖reward〗\_3=〖negativeReward〗\_1+〖negativeReward〗\_2
