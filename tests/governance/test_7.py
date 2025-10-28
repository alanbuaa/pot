import json
import socket
import logging
import struct

# 设置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def test_parameter_update():
    try:
        # 模拟参数数据
        test_data = {
            "action": "update",
            "level": "parameter",
            "group": "net",
            "param_name": "GossipSubD",
            "param_value": 100
        }
        
        # 使用socket替代Client
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(("127.0.0.1", 22326))
        
        # 转换为JSON字符串
        json_data = json.dumps(test_data).encode('utf-8')
        
        # 发送数据长度（4字节）
        length = len(json_data)
        sock.sendall(struct.pack('>I', length))  # 使用struct打包4字节整数
        
        # 发送实际数据
        sock.sendall(json_data)
        
        logger.info(f"测试数据已发送: {test_data}")
        sock.close()
        
    except ConnectionRefusedError:
        logger.error("无法连接到服务器，请确保服务器正在运行")
    except Exception as e:
        logger.error(f"测试过程中出现错误: {str(e)}")

if __name__ == "__main__":
    test_parameter_update() 