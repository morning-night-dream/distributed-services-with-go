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

レコード登録

```sh
curl -X POST -H "Content-Type: application/json" -d '{"record":{"value":"test"}}' http://localhost:8080
```

レコード取得

```sh
curl -X GET -H "Content-Type: application/json" -d '{"offset":1}' http://localhost:8080
```
