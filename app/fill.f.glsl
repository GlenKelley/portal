#version 150
uniform vec4 color;
uniform float depth;
out vec4 fragColor;
void main()
{
    fragColor = color;
    gl_FragDepth = depth;
}