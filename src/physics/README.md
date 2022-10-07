# Epsilon Physics Engine

This physics engine is a port of Bullet Physics SDK v3.24. Not all features or functionality are ported, and the layout 
is suitable for native and idiomatic Go. It is important to be aware that types are much more constrained due to language
differences. Int128 and Uint128 support is provided by [go-num by Blake Williams](https://github.com/shabbyrobe/go-num) 
as an inlined package.
