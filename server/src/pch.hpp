#ifndef PCH_HPP
#define PCH_HPP
// 预编译头: 尽量只放稳定且常用/体积大的头, 减少频繁改动触发全量重编译
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <inttypes.h>
#include <time.h>
#include <string>
#include <vector>
#include <map>
#include <algorithm>
// 体积较大的表达式库
#include "exprtk.hpp"
#endif // PCH_HPP
