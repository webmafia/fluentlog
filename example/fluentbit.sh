#!/bin/bash

# docker run -p 127.0.0.1:24224:24224 fluent/fluent-bit /fluent-bit/bin/fluent-bit \
# 	-i forward \
# 	-o stdout \
# 	-p format=json_lines \
# 	--verbose \
# 	-f 1


docker run -p 127.0.0.1:24224:24224 fluent/fluent-bit /fluent-bit/bin/fluent-bit -c flb.conf