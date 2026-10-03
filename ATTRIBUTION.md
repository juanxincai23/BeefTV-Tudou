# 引用与致谢声明（ATTRIBUTION）

> 本文件声明本项目的代码来源、授权基础与自有修改范围。分发或再发布本项目时，请连同本文件以及随附的 `LICENSE`、`NOTICE`、`THIRD_PARTY_NOTICES.md` 一并保留。

## 一、基础项目

本项目基于 GitHub 开源项目 **BeefTV** 构建并修改：

- 仓库地址：<https://github.com/glanderness/BeefTV>
- 授权协议：MIT License
- 原版权声明（按要求原样保留）：

```text
MIT License

Copyright (c) 2026 @beefnoode and BeefTV contributors
Copyright (c) 2026 basketikun
Copyright (c) 2026 ddcat

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

## 二、上游项目

BeefTV 官方 `NOTICE` 声明其代码衍生自 Infinite Canvas：

- 仓库地址：<https://github.com/basketikun/infinite-canvas>
- 基线版本：v0.5.0（commit `568f0f1`，后于 commit `890ba95` 改为 MIT 授权）

本项目同样受益于该上游项目，在此一并致谢。

## 三、本项目自有修改

在原版基础上，本项目截至 2026-10-03 做了以下修改：

1. 土豆（api.ai-tudou.net）图像协议适配：异步/同步图生图请求体字段对齐网关要求（`images` 数组），`tudou-image`、`tudou-image-sync` 插件相应调整；
2. 参考图出站归一化：`backend/internal/app/tudou_reference.go` 按 6MiB 预算直传或降采样重编码，避免网关静默丢弃大图；
3. 显式 @ 引用与连线参考图合并：土豆协议下未被 @ 的连线参考图按序追加（`isTudouImageProtocol` 能力开关，其他协议行为不变）；
4. 画布节点下载修复：下载前将失效的 `blob:`/`data:` 地址经鉴权缓存管线解析为可用地址；
5. 渠道预置与一键启动脚本（`一键启动BeefTV.bat` / `.command`）等分发辅助内容。
## 四、第三方组件

Wails、FFmpeg.wasm、Excalidraw、MediaPipe、React、Gin、GORM、SQLite 等依赖与捆绑资源仍归各自作者所有，适用其原有许可；经审计的依赖与资源清单见 `THIRD_PARTY_NOTICES.md`。文中出现的供应商与产品名称仅用于标识兼容的协议或服务，不构成隶属或背书关系。

## 五、免责声明

本项目与 BeefTV、Infinite Canvas 的原作者及贡献者不存在隶属或背书关系。软件按"现状"提供，不附带任何明示或默示的担保；对于因使用本项目（含上述自有修改）产生的任何主张、损害或其他责任，原作者与贡献者不承担义务，修改部分的作者亦仅在适用法律允许的最大范围内免责。
土豆AIQQ交流群 715943908 验证暗号“土豆土豆我是地瓜我是地瓜”
