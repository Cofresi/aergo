/**
 * @file    meta.h
 * @copyright defined in aergo/LICENSE.txt
 */

#ifndef _META_H
#define _META_H

#include "common.h"

#define TYPE_NAME(type)             type_strs_[type]

#define is_valid_type(type)         ((type) > TYPE_NONE && (type) < TYPE_MAX)

#define is_bool_meta(meta)          ((meta)->type == TYPE_BOOL)

#define is_integer_meta(meta)                                                  \
    ((meta)->type >= TYPE_INT8 && (meta)->type <= TYPE_UINT64)

#define is_float_meta(meta)                                                    \
    ((meta)->type == TYPE_FLOAT || (meta)->type == TYPE_DOUBLE)

#define is_string_meta(meta)        ((meta)->type == TYPE_STRING)

#define is_struct_meta(meta)        ((meta)->type == TYPE_STRUCT)
#define is_map_meta(meta)           ((meta)->type == TYPE_MAP)
#define is_ref_meta(meta)           ((meta)->type == TYPE_REF)
#define is_tuple_meta(meta)         ((meta)->type == TYPE_TUPLE)

#define is_const_meta(meta)         (meta)->is_const
#define is_array_meta(meta)         (meta)->is_array
#define is_untyped_meta(meta)       (meta)->is_untyped

#define is_primitive_meta(meta)     ((meta)->type <= TYPE_PRIMITIVE)
#define is_comparable_meta(meta)    ((meta)->type <= TYPE_COMPARABLE)

#define meta_set_prim               meta_set
#define meta_set_tuple(meta)        meta_set((meta), TYPE_TUPLE)
#define meta_set_void(meta)         meta_set((meta), TYPE_VOID)

#ifndef _META_T
#define _META_T
typedef struct meta_s meta_t;
#endif /* ! _META_T */

typedef enum type_e {
    TYPE_NONE       = 0,
    TYPE_BOOL,
    TYPE_BYTE,
    TYPE_INT8,
    TYPE_UINT8,
    TYPE_INT16,
    TYPE_UINT16,
    TYPE_INT32,
    TYPE_UINT32,
    TYPE_FLOAT,
    TYPE_INT64,
    TYPE_UINT64,
    TYPE_DOUBLE,
    TYPE_STRING,
    TYPE_STRUCT,
    TYPE_REF,
    TYPE_ACCOUNT,
    TYPE_COMPARABLE = TYPE_ACCOUNT,

    TYPE_MAP,
    TYPE_PRIMITIVE  = TYPE_MAP,

    TYPE_VOID,                      /* for return type of function */
    TYPE_TUPLE,                     /* for tuple expression */
    TYPE_MAX
} type_t;

typedef struct meta_map_s {
    type_t k_type;
    meta_t *v_meta;
} meta_map_t;

struct meta_s {
    type_t type;

    bool is_const;
    bool is_array;
    bool is_untyped;

    union {
        meta_map_t u_map;
    };
};

extern char *type_strs_[TYPE_MAX];

void meta_dump(meta_t *meta, int indent);

static inline void
meta_init(meta_t *meta)
{
    ASSERT(meta != NULL);

    memset(meta, 0x00, sizeof(meta_t));
}

static inline void
meta_set(meta_t *meta, type_t type)
{
    ASSERT(meta != NULL);
    ASSERT1(is_valid_type(type), type);

    meta->type = type;
}

static inline void
meta_set_untyped(meta_t *meta, type_t type)
{
    meta_set(meta, type);

    meta->is_untyped = true;
}

static inline void
meta_set_map(meta_t *meta, type_t k_type, meta_t *v_meta)
{
    meta_set(meta, TYPE_MAP);

    meta->u_map.k_type = k_type;
    meta->u_map.v_meta = v_meta;
}

static inline void
meta_set_from(meta_t *meta, meta_t *x, meta_t *y)
{
    ASSERT(meta != NULL);

    if (is_untyped_meta(x) && is_untyped_meta(y)) {
        ASSERT1(is_primitive_meta(x), x->type);
        ASSERT1(is_primitive_meta(y), y->type);

        meta_set_untyped(meta, MAX(x->type, y->type));
    }
    else if (is_untyped_meta(x)) {
        *meta = *y;
    }
    else {
        *meta = *x;
    }
}

static inline bool
meta_equals(meta_t *x, meta_t *y)
{
    if (is_untyped_meta(x) || is_untyped_meta(y)) {
        if (x->type == y->type ||
            (is_integer_meta(x) && is_integer_meta(y)) ||
            (is_float_meta(x) && is_float_meta(y)))
            return true;

        return false;
    }

    if (is_map_meta(x) || is_map_meta(y)) {
        if (is_ref_meta(x) || is_ref_meta(y) ||
            (x->type == y->type &&
             x->u_map.k_type == y->u_map.k_type &&
             meta_equals(x->u_map.v_meta, y->u_map.v_meta)))
            return true;

        return false;
    }

    return x->type == y->type;
}

#endif /* ! _META_H */