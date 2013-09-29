#version 150

uniform sampler2D textures[2];
uniform float glow;

in vec2 texcoord;
in float fade_factor;
in vec4 worldCoord;
in vec4 inceptionCoord;
out vec4 fragColor;

void main()
{
    fragColor = mix(
        texture(textures[0], texcoord),
        texture(textures[1], texcoord),
        fade_factor
    );
    vec3 i = inceptionCoord.xyz;
    //vec3 v = vec3(0.1,0.5,0.1) * inceptionCoord.xyz + vec3(0.5,0,0.5)
    vec3 v = clamp(sin(vec3(0.1,0.5,0.1) * i) + vec3(0.5,0,0.5),0,1);
    fragColor = mix(
        vec4(v, 1),
        vec4(0,1,0,1),
        glow);
    gl_FragDepth = mix(gl_FragCoord.z, 0, glow);
}