# qwmsg
## 说明
* 基于企业微信开放API的消息推送库
* 支持如下相关消息接口
    1. 上传临时素材
    2. 发送文字消息
    3. 发送图片消息
    4. 发送文件消息
    5. 发送卡片信息
    6. 发送图文信息
    7. 发送MD格式信息
## 特性
* 整个实例周期内Token自动刷新
* Token序列化到本地Cache文件中，避免重复创建、销毁实例带来的重复Token请求
## 示例

* 文字消息
> Corpid和Corpsecret当然是假的
```
ins := New(Config{Corpid: "qq46e2334254dasfwe", Corpsecret: "dwerfFEFFWEFE234324dwdffwrnmfljQ", Agentid: 1000002, Expiresin: 2400})
	ins.SendTextMsg("teststring\n232323232", false)
```