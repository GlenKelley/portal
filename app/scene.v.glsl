#version 150

uniform mat4 projection;
uniform mat4 cameraview;
uniform mat4 worldview;
uniform mat4 inception;
uniform mat4 portalview;
uniform float elapsed;

in vec3 position;
out vec2 texcoord;
out float fade_factor;
out vec4 worldCoord;
out vec4 inceptionCoord;

void main() {
    vec4 p = vec4(position, 1);
    worldCoord = worldview * p;
    inceptionCoord = inception * worldCoord;
    gl_Position = projection * cameraview * worldCoord;
    texcoord = position.xy * vec2(-0.5) + vec2(0.5);
    fade_factor = sin(elapsed)*0.5 + 0.5;
    gl_ClipDistance[0] = (portalview * worldCoord).z;
}
