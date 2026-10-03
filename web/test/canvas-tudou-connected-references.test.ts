import assert from "node:assert/strict";
import test from "node:test";

import "@/lib/canvas/node-registry/definitions";
import { buildNodeGenerationContext } from "@/components/canvas/canvas-node-generation";
import { CanvasNodeType, type CanvasConnection, type CanvasNodeData } from "@/types/canvas";

function imageNode(id: string, title: string): CanvasNodeData {
    return {
        id,
        type: CanvasNodeType.Image,
        title,
        position: { x: 0, y: 0 },
        width: 320,
        height: 180,
        metadata: { content: "data:image/png;base64,QUJD", mimeType: "image/png", status: "success" },
    };
}

function fixture() {
    const target: CanvasNodeData = {
        id: "gen-node",
        type: CanvasNodeType.Image,
        title: "生成节点",
        position: { x: 400, y: 0 },
        width: 320,
        height: 180,
        metadata: {},
    };
    const inputs = [imageNode("img-1", "图片一"), imageNode("img-2", "图片二"), imageNode("img-3", "图片三")];
    const nodes = [target, ...inputs];
    const connections: CanvasConnection[] = inputs.map((input, index) => ({ id: `c-${index}`, fromNodeId: input.id, toNodeId: target.id }));
    return { target, nodes, connections };
}

test("土豆协议：显式 @ 时未 @ 的连线图片一并送出（被 @ 的保持在前）", () => {
    const { target, nodes, connections } = fixture();
    const context = buildNodeGenerationContext(target.id, nodes, connections, "@图片1 的背景，让小狗们并排坐在一起", [], false, true);
    assert.deepEqual(
        context.referenceImages.map((image) => image.id),
        ["img-1", "img-2", "img-3"],
    );
});

test("其他协议：显式 @ 时保持只发被 @ 的图片", () => {
    const { target, nodes, connections } = fixture();
    const context = buildNodeGenerationContext(target.id, nodes, connections, "@图片1 的背景，让小狗们并排坐在一起", [], false, false);
    assert.deepEqual(
        context.referenceImages.map((image) => image.id),
        ["img-1"],
    );
});

test("没有 @ 时连线图片全部参与，两种开关结果一致", () => {
    const { target, nodes, connections } = fixture();
    for (const includeConnectedReferences of [false, true]) {
        const context = buildNodeGenerationContext(target.id, nodes, connections, "让小狗们并排坐在一起", [], false, includeConnectedReferences);
        assert.deepEqual(
            context.referenceImages.map((image) => image.id),
            ["img-1", "img-2", "img-3"],
        );
    }
});
