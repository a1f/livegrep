cc_library(
    name = "codesearch",
    srcs = glob([
        "*.cc",
    ]),
    hdrs = glob(["*.h"]),
    copts = ["-Wno-sign-compare"],
    visibility = ["//visibility:public"],
    deps = [
        "//src/lib",
        "//third_party:utf8cpp",
        "@boost//:filesystem",
        "@boost//:intrusive_ptr",
        "@com_github_json_c//:json",
        "@com_github_libgit2//:libgit2",
        "@com_github_sparsehash//:sparsehash",
        "@com_googlesource_code_re2//:re2",
        "@divsufsort",
    ],
)
