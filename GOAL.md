# GOAL

The aim of this project is to generate a short video to help the viewer gain an appreciation for the effects of special relativity.

The overall idea is to use relativistic ray tracing to render a video showing what it would look like to be a human moving in a familiar
environment, but where the speed of light is reduced to something at a more human scale (e.g. 1m/s).

The rendering should illustrate known effects such as length contraction, time dilation and redshift--but these effects should all be
produced native to the ray tracing and not somehow calculated or added separately.

I have in mind a first-person view from a person walking through a living room type of environment (with sunlight coming in through a window,
perhaps a mirror and a glass globe furnishing (art on a coffee table), various stationary furniture items, and perhaps a cat or another human
moving through the environment.

Objects should possess textures including full spectrum reflectivity/emission characteristics so that redshift effects can be rendered properly.

The project should start with a simple scene, to get it rendering properly, and then add more objects and effects.

It should be done in a modular way so that it can be built up over time.

It is intended to be rendered on a machine with 64 cores, 256GB of memory and a GPU, so the code should take advantage of these available resources.

I have in mind that it be written in Go, but other languages can be considered.
