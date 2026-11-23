#version 440
// Adaptado do Ryotunes (GPL-3.0-or-later). Para recompilar:
//   qsb --glsl "100 es,120,150" --hlsl 50 --msl 12 -o rounded.frag.qsb rounded.frag
// Rounded-rectangle clip for a single Image layer: one quad, one texture read, no mask source.
// `radius` and `size` are in the layer's pixels; a 1 px feather keeps the edge anti-aliased.
layout(location = 0) in vec2 qt_TexCoord0;
layout(location = 0) out vec4 fragColor;
layout(std140, binding = 0) uniform buf {
    mat4 qt_Matrix;
    float qt_Opacity;
    vec2 size;
    float radius;
};
layout(binding = 1) uniform sampler2D source;

void main() {
    vec2 p = qt_TexCoord0 * size;
    vec2 q = abs(p - size * 0.5) - (size * 0.5 - vec2(radius));
    float d = length(max(q, 0.0)) + min(max(q.x, q.y), 0.0) - radius;
    float a = 1.0 - smoothstep(-1.0, 0.0, d);
    fragColor = texture(source, qt_TexCoord0) * a * qt_Opacity;
}
