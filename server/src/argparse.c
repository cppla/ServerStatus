#include "argparse.h"

#if defined(__cplusplus)
extern "C" {
#endif

#define OPT_UNSET 1

static const char *
prefix_skip(const char *str, const char *prefix)
{
    size_t len = strlen(prefix);
    return strncmp(str, prefix, len) ? NULL : str + len;
}

int
prefix_cmp(const char *str, const char *prefix)
{
    for (;; str++, prefix++)
        if (!*prefix)
            return 0;
        else if (*str != *prefix)
            return (unsigned char)*prefix - (unsigned char)*str;
}

static void
argparse_error(struct argparse *this_, const struct argparse_option *opt,
               const char *reason)
{
    if (!strncmp(this_->argv[0], "--", 2)) {
        fprintf(stderr, "error: option `%s` %s\n", opt->long_name, reason);
        exit(-1);
    } else {
        fprintf(stderr, "error: option `%c` %s\n", opt->short_name, reason);
        exit(-1);
    }
}

static int
argparse_getvalue(struct argparse *this_, const struct argparse_option *opt,
                  int flags)
{
    const char *s = NULL;
    if (!opt->value)
        goto skipped;
    switch (opt->type) {
    case ARGPARSE_OPT_BOOLEAN:
        if (flags & OPT_UNSET) {
            *(int *)opt->value = *(int *)opt->value - 1;
        } else {
            *(int *)opt->value = *(int *)opt->value + 1;
        }
        if (*(int *)opt->value < 0) { 
            *(int *)opt->value = 0;
        }
        break;
    case ARGPARSE_OPT_BIT:
        if (flags & OPT_UNSET) {
            *(int *)opt->value &= ~opt->data;
        } else {
            *(int *)opt->value |= opt->data;
        }
        break;
    case ARGPARSE_OPT_STRING:
        if (this_->optvalue) {
            *(const char **)opt->value = this_->optvalue;
            this_->optvalue = NULL;
        } else if (this_->argc > 1) {
            this_->argc--;
            *(const char **)opt->value = *++this_->argv;
        } else {
            argparse_error(this_, opt, "requires a value");
        }
        break;
    case ARGPARSE_OPT_INTEGER:
        if (this_->optvalue) {
            *(int *)opt->value = strtol(this_->optvalue, (char **)&s, 0);
            this_->optvalue = NULL;
        } else if (this_->argc > 1) {
            this_->argc--;
            *(int *)opt->value = strtol(*++this_->argv, (char **)&s, 0);
        } else {
            argparse_error(this_, opt, "requires a value");
        }
        if (*s)
            argparse_error(this_, opt, "expects a numerical value");
        break;
    default:
        assert(0);
    }

skipped:
    if (opt->callback) {
        return opt->callback(this_, opt);
    }

    return 0;
}

static void
argparse_options_check(const struct argparse_option *options)
{
    for (; options->type != ARGPARSE_OPT_END; options++) {
        switch (options->type) {
        case ARGPARSE_OPT_END:
        case ARGPARSE_OPT_BOOLEAN:
        case ARGPARSE_OPT_BIT:
        case ARGPARSE_OPT_INTEGER:
        case ARGPARSE_OPT_STRING:
            continue;
        default:
            fprintf(stderr, "wrong option type: %d", options->type);
            break;
        }
    }
}

static int
argparse_short_opt(struct argparse *this_, const struct argparse_option *options)
{
    for (; options->type != ARGPARSE_OPT_END; options++) {
        if (options->short_name == *this_->optvalue) {
            this_->optvalue = this_->optvalue[1] ? this_->optvalue + 1 : NULL;
            return argparse_getvalue(this_, options, 0);
        }
    }
    return -2;
}

static int
argparse_long_opt(struct argparse *this_, const struct argparse_option *options)
{
    for (; options->type != ARGPARSE_OPT_END; options++) {
        const char *rest;
        int opt_flags = 0;
        if (!options->long_name)
            continue;

        rest = prefix_skip(this_->argv[0] + 2, options->long_name);
        if (!rest) {
            // Negation allowed?
            if (options->flags & OPT_NONEG) {
                continue;
            }
            // Only boolean/bit allow negation.
            if (options->type != ARGPARSE_OPT_BOOLEAN && options->type != ARGPARSE_OPT_BIT) {
                continue;
            }

            if (!prefix_cmp(this_->argv[0] + 2, "no-")) {
                rest = prefix_skip(this_->argv[0] + 2 + 3, options->long_name);
                if (!rest)
                    continue;
                opt_flags |= OPT_UNSET;
            } else {
                continue;
            }
        }
        if (*rest) {
            if (*rest != '=')
                continue;
            this_->optvalue = rest + 1;
        }
        return argparse_getvalue(this_, options, opt_flags);
    }
    return -2;
}

int
argparse_init(struct argparse *this_, struct argparse_option *options,
              const char *usage, int flags)
{
    memset(this_, 0, sizeof(*this_));
    this_->options = options;
    this_->usage = usage;
    this_->flags = flags;
    return 0;
}

int
argparse_parse(struct argparse *this_, int argc, const char **argv)
{
    this_->argc = argc - 1;
    this_->argv = argv + 1;
    this_->out = argv;

    argparse_options_check(this_->options);

    for (; this_->argc; this_->argc--, this_->argv++) {
        const char *arg = this_->argv[0];
        if (arg[0] != '-' || !arg[1]) {
            if (this_->flags & ARGPARSE_STOP_AT_NON_OPTION) {
                goto end;
            }
            // if it's not option or is a single char '-', copy verbatimly
            this_->out[this_->cpidx++] = this_->argv[0];
            continue;
        }
        // short option
        if (arg[1] != '-') {
            this_->optvalue = arg + 1;
            switch (argparse_short_opt(this_, this_->options)) {
            case -1:
                break;
            case -2:
                goto unknown;
            }
            while (this_->optvalue) {
                switch (argparse_short_opt(this_, this_->options)) {
                case -1:
                    break;
                case -2:
                    goto unknown;
                }
            }
            continue;
        }
        // if '--' presents
        if (!arg[2]) {
            this_->argc--;
            this_->argv++;
            break;
        }
        // long option
        switch (argparse_long_opt(this_, this_->options)) {
        case -1:
            break;
        case -2:
            goto unknown;
        }
        continue;

unknown:
        fprintf(stderr, "error: unknown option `%s`\n", this_->argv[0]);
        argparse_usage(this_);
        exit(0);
    }

end:
    memmove(this_->out + this_->cpidx, this_->argv,
            this_->argc * sizeof(*this_->out));
    this_->out[this_->cpidx + this_->argc] = NULL;

    return this_->cpidx + this_->argc;
}

void
argparse_usage(struct argparse *this_)
{
    fprintf(stdout, "Usage: %s\n", this_->usage);
    fputc('\n', stdout);

    const struct argparse_option *options;

    // figure out best width
    size_t usage_opts_width = 0;
    size_t len;
    options = this_->options;
    for (; options->type != ARGPARSE_OPT_END; options++) {
        len = 0;
        if ((options)->short_name) {
            len += 2;
        }
        if ((options)->short_name && (options)->long_name) {
            len += 2;           // separator ", "
        }
        if ((options)->long_name) {
            len += strlen((options)->long_name) + 2;
        }
        if (options->type == ARGPARSE_OPT_INTEGER) {
            len += strlen("=<int>");
        } else if (options->type == ARGPARSE_OPT_STRING) {
            len += strlen("=<str>");
        }
        len = ceil((float)len / 4) * 4;
        if (usage_opts_width < len) {
            usage_opts_width = len;
        }
    }
    usage_opts_width += 4;      // 4 spaces prefix

    options = this_->options;
    for (; options->type != ARGPARSE_OPT_END; options++) {
        size_t pos;
        int pad;
        pos = fprintf(stdout, "    ");
        if (options->short_name) {
            pos += fprintf(stdout, "-%c", options->short_name);
        }
        if (options->long_name && options->short_name) {
            pos += fprintf(stdout, ", ");
        }
        if (options->long_name) {
            pos += fprintf(stdout, "--%s", options->long_name);
        }
        if (options->type == ARGPARSE_OPT_INTEGER) {
            pos += fprintf(stdout, "=<int>");
        } else if (options->type == ARGPARSE_OPT_STRING) {
            pos += fprintf(stdout, "=<str>");
        }
        if (pos <= usage_opts_width) {
            pad = usage_opts_width - pos;
        } else {
            fputc('\n', stdout);
            pad = usage_opts_width;
        }
        fprintf(stdout, "%*s%s\n", pad + 2, "", options->help);
    }
}

int
argparse_help_cb(struct argparse *this_, const struct argparse_option *option)
{
    (void)option;
    argparse_usage(this_);
    exit(0);
    return 0;
}

#if defined(__cplusplus)
}
#endif
