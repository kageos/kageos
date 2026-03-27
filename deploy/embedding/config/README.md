# Embedding — 配置说明

官方配置入口已调整为 **`deploy/dev/config/`** 与 **`deploy/prod/config/`**。

Embedding 裸机场景当前仍兼容使用 **`deploy/config/prod/`** 作为本机 prod 覆盖目录。

本机覆盖请使用 **`deploy/config/local/`**，并在仓库根执行：

```bash
bash deploy/embedding/scripts/embedding.sh local
```

详见 **[../../deploy/config/README.md](../../deploy/config/README.md)**。
