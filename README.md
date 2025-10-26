# Mistnet Signaling Server

English version: [README.en.md](./README.en.md)

これは、WebRTCアプリケーション向けのシグナリングサーバーです。

Go言語で実装されており、クライアント（ノード）が部屋（Room）に参加し、同じ部屋にいる他のクライアントとPeer-to-Peer接続を確立するのを助けます。

## 主な機能

- WebSocketによるリアルタイム通信
- ルームベースのクライアント管理
- ルーム参加時の自動的なピア検出とペアリング要求
- 設定ファイル (`config.json`) による簡単な設定
- `zap` と `lumberjack` を利用したログローテーション機能

## 使用方法

### 1. 前提条件

- Go (1.18以上を推奨) がインストールされていること。

### 2. ビルド

リポジトリをクローンし、以下のコマンドでビルドします。

```sh
go build
```

### 3. 設定

サーバーを初めて実行すると、カレントディレクトリに `config.json` ファイルが自動的に生成されます。

```json
{
  "GlobalNode": {
    "Enable": true,
    "Port": 8080
  }
}
```

- **Enable**: `true` に設定すると、サーバーが起動します。
- **Port**: サーバーがリッスンするポート番号を指定します。

### 4. サーバーの実行

ビルドして生成された実行ファイルを起動します。

```sh
./mistnet-signaling
```

サーバーが `config.json` で指定されたポート（デフォルト: 8080）で起動します。ログは `./logs/app.log` に出力されます。

## プロトコル仕様

### エンドポイント

シグナリングサーバーは、以下のWebSocketエンドポイントを提供します。

`ws://<server_address>:<port>/signaling`

### メッセージ形式

クライアントとサーバー間の通信は、以下のJSON形式のメッセージで行われます。

```json
{
  "Type": "Request",
  "Data": "...",
  "SenderId": "node-id-of-sender",
  "ReceiverId": "node-id-of-receiver",
  "RoomId": "room-id"
}
```

- **Type**: メッセージの種類。
  - `Request`: クライアントがルームに参加する際や、サーバーがクライアントにP2P接続の開始を要求する際に使用します。
  - その他（`offer`, `answer`, `candidate`など）: これらのメッセージは、サーバーによって指定された `ReceiverId` のクライアントにそのまま転送されます。
- **Data**: メッセージのペイロード（例: SDPオファー/アンサー、ICE候補など）。
- **SenderId**: メッセージを送信するクライアントの一意なID。
- **ReceiverId**: メッセージを受信するクライアントの一意なID。サーバーへの`Request`メッセージの場合、このフィールドは空でも構いません。
- **RoomId**: クライアントが参加するルームのID。

### 通信フロー

1.  **接続とルーム参加**
    クライアントはWebSocketエンドポイントに接続し、ルームに参加するために `Type: "Request"` のメッセージを送信します。

    **Client A -> Server:**
    ```json
    {
      "Type": "Request",
      "SenderId": "node-A",
      "RoomId": "room-123"
    }
    ```

2.  **ピアのペアリング要求**
    ルームに新しいクライアントが参加すると、サーバーはルーム内の他のピアとの接続を促すために、`Type: "Request"` のメッセージを関係するクライアントに送信します。例えば、Client Bが同じルームに参加すると、サーバーはAとBに相互接続を要求します。

    **Server -> Client A:**
    ```json
    {
      "Type": "Request",
      "SenderId": "node-B", // 接続相手のID
      "ReceiverId": "node-A",
      "RoomId": "room-123"
    }
    ```
    **Server -> Client B:**
    ```json
    {
      "Type": "Request",
      "SenderId": "node-A", // 接続相手のID
      "ReceiverId": "node-B",
      "RoomId": "room-123"
    }
    ```
    このメッセージを受信したクライアントは、`SenderId` を相手としてWebRTCの接続処理（Offerの生成など）を開始します。

3.  **WebRTCシグナリングメッセージの転送**
    SDPのOffer/AnswerやICE CandidateなどのWebRTCシグナリングメッセージは、`ReceiverId` を指定してサーバーに送信します。サーバーはこれらのメッセージを該当クライアントにそのまま転送します。

    **Client A -> Server (Offerを転送):**
    ```json
    {
      "Type": "offer",
      "Data": "{ "sdp": "..." }",
      "SenderId": "node-A",
      "ReceiverId": "node-B",
      "RoomId": "room-123"
    }
    ```
    サーバーはこのメッセージをClient Bに転送します。

4.  **切断**
    クライアントがWebSocket接続を切断すると、サーバーはそのクライアントをルームから自動的に削除します。

## ライセンス

このプロジェクトは `LICENSE` ファイルに記載されたライセンスの下で公開されています。
