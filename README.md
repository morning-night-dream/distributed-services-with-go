# distributed-services-with-go

[Go 言語による分散サービス](https://www.oreilly.co.jp/books/9784873119977/)

# ログの構築

- レコード：ログに保存されるデータ
- ストア：レコードを保存するファイル
- インデックス：インデックスエントリを保存するファイル
- セグメント：ストアとインデックスをまとめているものの抽象的概念
- ログ：セグメントをすべてまとめているものの抽象的概念

# 動作確認

起動

```sh
make run
```

## HTTP サーバー経由

レコード登録

```sh
curl -X POST -H "Content-Type: application/json" -d '{"record":{"value":"test"}}' http://localhost:8080
```

レコード取得

```sh
curl -X GET -H "Content-Type: application/json" -d '{"offset":1}' http://localhost:8080
```

## gprc サーバー経由

レコード登録

```sh
grpcurl -cert $HOME/.proglog/client.pem -key $HOME/.proglog/client-key.pem -cacert $HOME/.proglog/ca.pem  -import-path ./api/v1 -proto log.proto -d '{"record":{"value": "01011101"}}' localhost:8081 log.v1.LogService/Produce
```

レコード取得

```sh
grpcurl -cert $HOME/.proglog/client.pem -key $HOME/.proglog/client-key.pem -cacert $HOME/.proglog/ca.pem -import-path ./api/v1 -proto log.proto -d '{"offset":1}' localhost:8081 log.v1.LogService/Consume
```

※事前に`make gencert`を実行すること
