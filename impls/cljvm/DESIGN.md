# BasiLISC design notes

> Basic Lisp for Small Computers

This system is intended to be portable to small computers, ideally to 8- and
16-bit machines. Some machines might be able to support self-hosting, but any
machine can be targeted by the cross-compiler.

The Clojure program contains several components:
- **Interpreter:** A BasiLISC-compatible system in native Clojure, using Clojure native data. (Complete)
- **Hosted VM:** A BasiLISC-compatible system powered by the VM, using Clojure native data. (Incomplete)
    -  Main purpose is testing the VM basics.
- **Reference VM:** A BasiLISC system written in Clojure, using low-level machine data. (Incomplete)
    - This is a complete BasiLISC cross-compiler target that happens to be
      written in Clojure.
    - This serves several purposes: as a testbed and reference implementation,
      but also it implements macros during cross-compilation.

# Cross-compilation process

There are three vital inputs to the compiler:
- 1 or more Lisp text files
- The `:live` flag:
    - If true, Lisp code for the reader, compiler, etc. are included in the
      compilation. Then the target system includes a REPL. The code for
      functions etc. is placed in live GC'd memory.
    - If false, only the application code and core library is compiled. All
      (non-atom) data and functions in the namespaces are placed in tenured,
      permanent memory, since it can't be replaced. Data referenced only by
      atoms goes in live memory.
- The `:target` machine. This is the dispatch value of some multimethods that
  control how things are stored on various target machines.


1. The Interpreter's reader is used to read the Mal text into Clojure data.
2. The namespaces, functions and data are compiled into the binary formats.
3. A BasiLISC binary is compiled. Exactly how the compiled segments of data are
   placed into that binary is target-dependent.
4. Native code to implement the VM, garbage collector, and primitive operations
   needs to be linked in somehow. That might be done statically for a simple
   retrocomputer, or by `ld` or similar for a modern machine.


# VM execution model

The VM stores its working data in two (logical) stacks stored in the heap. The
heap is divided into 4 unequal spaces, and managed by a garbage collector.
Compiled methods are stored as a vector, with the first several cells reserved
and then bytecode packed into the cells.

## The stacks

First, the **call stack** - a linked list of *activation records* corresponding
to each Lisp call. These are stored like vectors, with the first several cells
reserved for specific pointers, and the rest reserved for the data stack.

Second is the **data stack** - the tail of each activation record holds some
cells, which are used to hold temporary values for calls. The function call
bytecode gives the argcount, and that many cells from the caller's stack are
copied to use as args in the callee. The first argument is deepest on the stack,
and the last argument is on top.

Functions specify their fixed parameter count, and whether they allow variadic
arguments. If there are any variadic args, the `:call` or `:dyncall` opcodes
will build a `list` in the heap, popping the arguments into it (so it's handy
they're in this order). Then a vector is created with `fixed_params` cells, plus
1 if it's variadic. This is filled right to left with popped values, with the
varargs list in the rightmost slot.


## Function calls

This is obviously a vital, bread-and-butter operation in a Lisp VM!

There are several callable values: named Lisp functions, built-in functions,
anonymous closure functions, maps, sets, and keywords. When compiling a
function call with a literal symbol, the compiler will search the environment
for that symbol.

- If it's in the local environment, it's dynamic, so a `dyncall` opcode is
  generated.
- If it's a compile-time literal (eg. a set or map with only literal values),
  it gets assembled into the literals array for this function. Then it can be
  called with a `call` opcode.
- If it's a run-time literal (eg. a map or set in literal syntax but containing)
  dynamic values, code to generate it is compiled first, then the code for the
  arguments, and finally the `dyncall`.
- If it's a function in a namespace, the symbol is put in the literals array and
  a `call` is compiled.

Let's consider first a Lisp function like this:
`(fn [x y] (sqrt (* x x) (+ y 1)))`. (Let's suppose `sqrt` is a Lisp function,
and `+` and `*` are primitives.)

All three of these functions can be found in the global namespaces, so pointers
to their `Var`s are put in the literals array, along with `1`:
`[#'sqrt #'+ #'* 1]`. Args and locals go in another list, with the args first,
so `x` is `0` and `y` is `1`.

Then the bytecode for the function body would look like this:

```clojure
             ;; Data stack:
[:local 0]   ;; x
[:local 0]   ;; x         x
[:call 2 1]  ;; x-squared
[:local 1]   ;; x-squared y
[:literal 3] ;; x-squared y   1
[:call 2 2]  ;; x-squared y+1
[:call 2 0]  ;; square-root
[:return]
```

You can see how the stack plays nicely with "applicative order" evaluation.

Dynamic calls with `:dyncall` bytecodes expect the function to be deepest on
the stack - the `:dyncall` peeks at it, handles the args, then pops the callee
and checks its type.

### Activation records

Each Lisp call corresponds to an activation record in the heap. These each point
to their caller, forming a linked list - logically this is a call stack. An
activation record is stored as a vector, where the first handful of slots are
used for its own data, and then the tail is the data stack for the call.

Those reserved slots are:
- `caller` - points to the caller's activation record
- `bytecode` - pointer to the bytecode "vector"
- `current` - *byte offset* into the `bytecode` array where the *next* bytecode
  to execute is found (this should fit into a small integer)
- `literals` - pointer to a vector of literal values (or nil if there are none)
- `context` - pointer to a vector holding the args and any locals (ie. `let`
  bindings)
- `enclosure` - pointer to a vector of enclosed values (nil for top-level functions)
- `sp` - index into the stack (index into this entire array, not just the stack portion)
- (... stack values ...)

### Let bindings

The `context` array on an activation record looks like this:

```
arg0 arg1 arg2 ... argN argRest local0 local1 ...
```

`argRest` points to the list of varargs; it's not present if there are only
fixed parameters.

There are as many `localN` slots as there are `let` bindings in the function
(outside any inner functions, see below). Note that shadowed bindings, ie. when
two different bindings have the same name, they get *different* slots in the
locals list.

### Multiple arities

BasiLISC supports multiple arities, like Clojure. Each arity has separate code,
so it compiles a separate bytecode array. The data structure for a function
looks like this:

```
literals          pointer to an array of literals - shared by all arities
enclosure         pointer to array of values captured by this closure (or nil)
context_size      number of slots for args and locals - max across all arities
arity_a_width     number of fixed args for the smallest fixed arity
arity_a_code      pointer to the bytecode for smallest fixed arity
arity_b_width     number of fixed args for the second smallest fixed arity
arity_b_code      pointer to the bytecode for second smallest fixed arity
...
arity_m_width     number of fixed args for the largest fixed arity
arity_m_code      pointer to the bytecode for largest fixed arity
variadic_width    negated number of total args for the single variadic arity
variadic_code     pointer to the bytecode for the variadic arity
```

Note that there can only be one variadic arity, since two variadic arities can't
be told apart. The fixed arities (if any) appear in ascending order.

If there is no variadic arity, the `variadic_width` slot is filled with -1 and
`variadic_code` with `nil`. That simplifies the search through the arities
performed by the calling code - when it finds a negative width, it can stop. If
the `variadic_code` is `nil`, throw an arity mismatch; if it's a real pointer
then the width is correct.


### Closures

Inner functions are more complicated than a top-level function because they can
"close" over bindings from the scope where they're defined, including dynamic
things like the results of function calls, or the arguments to the outer
function.

The compiler will generate function objects (see below) for them in the same way
as for any other function, and put these in the literals array. Then the code
for the function looks like this:

```
(code to get the value of enclosure 0 on the stack)
(code to get the value of enclosure 1 on the stack)
...
(code to get the value of enclosure N-1 on the stack)
[:closure N literal]
```

The `:closure` opcode will then create an `N`-element array, draining the stack
into it. (This freezes the current values permanently, so any `recur` or other
mutations in the outer function's state don't impact it.) It then *copies* the
function whose literal number is given, replacing its `enclosure` field. This
new function value is pushed onto the stack.

Closures do not retain the activation record that created them, so they can
outlive the call or even be passed between threads without fear.

If there are no enclosures, the compiler will simplify by putting the function
value in the literals array and using `[:literal i]` to push it or `[:call i]`
to call it statically.


# Bytecodes

There are two forms of the bytecode: as Clojure vectors, and a proper byte
encoding. The Clojure vector form is only used by the Hosted VM.

A summary of the bytecodes:
- Extended bytecode (special)
- Read arg/local N
- Read literal N
- Read enclosure N
- Call a function by its Var
- Dynamically call a function on the stack
- Set a binding
- Create a closure
- Call a primitive operation
- Push a common value
- VM operations

That's 11, so let's call it 4 bits each for the opcode and data value:
`oooodddd`. Some bytecodes use following bytes `x` and `y`. There's space for
a few more opcodes if they prove useful.

## 0 - extended

This is just VM machinery. The true opcode is in `d`, and the value normally
stored in `d` is in the next byte. `x` and `y` follow, if any.

This is used when there are too many literals or values in the environment
to fit the value in 4 bits (16 values).

(No Clojure form for this one; it only exists for encoding.)

## 1 - read context

`d` gives the index into the `context` array. The context array looks like
`arg0 ... argN argRest local0 ... localN`. (There's no `argRest` for a fixed
arity.)

Clojure: `[:local i]`

## 2 - read literal

Reads index `d` of the `literals` array for this function (stored in the
function value, and copied into activation records).

Clojure: `[:literal i]`

## 3 - read enclosure

Reads index `d` of the `enclosure` array for this function (stored in the
function value, and copied into activation records).

Clojure: `[:enclosure i]`

## 4 - call

Calls the function whose Var or function value is in literal `x` with `d`
arguments. These arguments should already be on the stack; see the function
call process above.

Clojure: `[:call argc literal-index]`

## 5 - dynamic call

Calls the function (or function-like value) found underneath its `d` arguments
on the stack. This works similarly to `call` but expects the function value to
be on the stack. Peeks at the function value on the stack, and resolves it to a
proper function value.

Clojure `[:dyn-call argc]`

## 6 - set binding

Pops the top value off the stack and stores it into slot `d` in the `context`.
This is a mutation operation, but the compiler should only generate this once
for each local slot.

Note that, even if the `context` array and activation record have been GC'd and
now live in old space, no "write barrier" check is needed for this store.
Since all live threads (and their call stacks) are already GC roots, the context
will already be crawled by the GC.

Clojure `[:bind local-index]`

## 7 - create closure

Constructs a closure value. The enclosed values should already have been pushed
(the compiler will generate code for this) and `d` holds the enclosure count.
`x` holds the literal number for the compiled function.
See "Closures" above.

Clojure: `[:closure enclosed-count literal-index]`

## 8 - primitive operation

`d` gives the arg count, and `x` the number of a primitive operation.
(For speed, when the compiler sees a static call to a function implemented on
the machine rather than in Lisp, it can generate this opcode instead of pushing
a literal and using `call`.)

(The stack effect of a primitive depends on the primitive - most drain all the
arguments and leave a return value, like a function call, but not necessarily.)

Clojure: `[:primitive argc primitive-number]`

## 9 - common values

To save space in compiled code, many commonly referenced literal values can be
pushed to the stack using this opcode.

| `d` | Value |
| :-- | :-- |
| 0-9 | Integers 0-9 |
| 10  | -1 |
| 11  | `nil` |
| 12  | `true` |
| 13  | `false` |
| 14  | `[]` |
| 15  | `{}` |

Note that there's room for lots more using the "extended" opcode, but I can't
think of anything except loading in more numbers.

## 15 - specials

The special operations are little extra VM operations. `d` is treated as an
extra opcode.

| `d` | Clojure | Operation |
| :-- | :-- | :-- |
| 0 | `[:dup]` | Duplicate top of stack |
| 1 | `[:drop]` | Discard top of stack |
| 2 | `[:return]` | Return top of stack to caller |
| 3 | `[:skip-when x]` | Pop top of stack. If truthy, skip forward by `x` bytes. |
| 4 | `[:skip-not x]` | Pop top of stack. If falsy, skip forward by `x` bytes. |
| 5 | `[:skip x]` | Unconditionally skip forward by `x` bytes. |


# Memory model

Generational garbage collection for all live data. Memory is divided into six spaces:

- Static code and data (eg. the code for the GC and primitive operations).
  Fixed and mostly read-only.
- Native CPU stack - Very small, only used during GC runs and primitive code.
- **Tenured:** Values which survive enough major collections are assumed to
  live forever; they get copied here.
- **Old:** The survivors of previous collections.
- **Reserve:** The space between OldSpace and Eden. Size is equal to Eden.
- **Eden:** Large area; new values are allocated here.

Tenured space, Old, Reserve and Eden must all be contiguous in this model!
See the Garbage Collection section for more about how the collection proceeds.

Let *b* be the number of bits in a pointer on the host machine. A **cell** is
a *b*-bit number. A Lisp value is stored as a contiguous block of 2 or more
cells. The first (0th?) cell of every value is the **header**, described below.

A cell can hold three things:
- a **small integer**,
- a **pointer to a value**, or
- *b/8* bytes of a string

## Value header

All values in memory have a header in their first cell. This contains several
bit fields, which indicate the type, the GC count, and so on.

A header value of 0 is used by the GC to signal that a value has already been
copied, so it must be arranged that the header is never otherwise 0. This is
handled simply: the type code cannot be 0.

The header has a type code in its low bits and a GC count in its high bits.
The space is not used very efficiently, but this is partly by design. Even on
a 16-bit machine there are plenty of bits available.

The header looks like this:
```
gggggggg ... tttttttt
```
where `g` is the GC counter and `t` is the type code. Each are 8 bits long even
though in practice there are only about a dozen types, and the tenuring
threshold is likely to be under 8. See the next section for the types.

### Optimizations in the header

On following a pointer to a value, the GC needs to know two things: has this
pointer already been forwarded, and has it reached the tenuring threshold.
Many processors will populate the condition codes on reading the header from
memory, which makes the frequent `forwarded?` check free.

Similarly, if the starting `g` value is chosen as `0x80 - TenureThreshold`,
then when the count is incremented, a value due for tenuring will have its
top bit set, ie. the header will become negative.

## Data types

This section details how each type is stored in memory, indexed by their type
code.

A general principle behind the numbering: several of the types are different to
the Lisp runtime, but can be treated identically by the GC. These are grouped
by bit pattern, allowing the GC to mask out a few bits of the type, and treat
several types identically.

Therefore the four two-pointer types are grouped together (`t = 000011xx`), and
so are the four opaque-bytes types (`t = 000010xx`).

| Code | Group  | Type |
| :--  | :--    | :--  |
| (0)  | --     | Small integer |
| 1    | --     | Nil |
| 2    | --     | Boolean |
| 3    | --     | Vector |
| 4    | --     | Node |
| 5-7  | --     | (reserved) |
| 8    | String-like `10xx` | String |
| 9    | String-like `10xx` | Symbol |
| 10   | String-like `10xx` | Keyword |
| 11   | String-like `10xx` | Compiled code |
| 12   | 2-cell `11xx` | List |
| 13   | 2-cell `11xx` | Map |
| 14   | 2-cell `11xx` | Env |
| 15   | 2-cell `11xx` | Atom |


### Type 0 - Small integers

Small integers are *(b-1)*-bit 2's complement integers. They are stored in the
upper bits of a cell, leaving the least-significant bit free - it is always 1.
Small integers do not get allocated on the heap as separate values - instead
they are stored directly in the cells of other values.

Note that the type code for small integers is never used in a header, since
they are stored into cells rather than pointed to. However since the type code
0 is reserved (see the Header section) it makes sense to use it for small
integers, for things like indexing a table of routines by the type code.

### Type 1 - Nil

There's only one `nil` in static space, but it's special, so it gets its own
code.

### Type 2 - Boolean

`true` and `false` are static values outside of GC.

### Type 3 - Vector

```
+--------+------+---------+---------+---------+---------+-----+
| header | size | index 0 | index 1 | index 2 | index 3 | ... |
+--------+------+---------+---------+---------+---------+-----+
```

Note that `size` is **the number of cells in the whole allocation** - it's not
the number of elements in the vector. The length of the vector is `size - 2`
cells. (The GC needs this size much more often than vectors' lengths are
checked while running Lisp code.)

Note that several internal structures (like activation records) are stored as
vectors with a specific size.

### Type 4 - Node (in a map)

Fixed size of 5:

```
+--------+--------+--------+--------+--------+
| header | key    | value  | left   | right  |
+--------+--------+--------+--------+--------+
```

### Types 8, 9 and 10 - String-likes

From a memory management point of view, all four of these are identical.

```
+--------+--------+---------+------------------------------+
| header | length | pointer | UTF-8 encoded string data    |
+--------+--------+---------+------------------------------+
```

The `length` is the number of bytes in the string. The GC has to compute the
allocation size by adding the size of 3 cells, then round that up to a whole
cell. Let `2^S = B`, where `B` is the number of bytes in a cell, and `S` is the
corresponding number of bits. Then `size = (length + 4B - 1) >> S`.
(On a 32-bit machine, `size = (length + 15) >> 2`;
on 64-bit `size = (length + 31) >> 3`.)

The `pointer` is for namespaced keywords and symbols. It points to another
symbol or keyword giving the namespace portion. That is, a keyword like
`:foo/bar` would be stored as:

```
+--------+--------+---------+----------+
| header | 3      |    |    | b a r    |
+--------+--------+----|----+----------+
                       |
    +------------------+
    |
+---v----+--------+---------+----------+
| header | 3      | nil     | f o o    |
+--------+--------+---------+----------+
```


### Type 11 - Compiled code

The `length` is the number of bytecodes; `pointer` holds the literal vector.
Otherwise these are stored exactly like strings. There's no end marker for
the code of a function; the compiler will generate a return.

### Types 12 through 15 - Fixed 2-cell values

```
      +--------+-------+----------+
List: | header | head  | tail     |
      +--------+-------+----------+
Map:  | header | root  | size     |
      +--------+-------+----------+
Env:  | header | map   | parent   |
      +--------+-------+----------+
Atom: | header | value | watchers |
      +--------+-------+----------+
```


## Garbage collection

As noted above, the GC "arena" is a contiguous block of address space, of size
*M*. The arena is divided into four spaces, like this (`X` used, ` ` free):
```
  Tenured   Old       Reserve            Eden
+----------------------------------------------------------+
|XXXXXXXXX|XXXXXXXXX|                  |              XXXXX|
+----------------------------------------------------------+
```

Reserve and Eden are always the same size, since after a major collection the
free space is split between them. (If there's an odd amount of cells, the extra
one goes to Reserve.) This guarantees that even if everything in Eden survives
a minor collection (which is a very pathological case) there will be room for
it in Reserve so the GC won't crash.

### Lifecycle 1 - Allocation

New memory is allocated at the top (high addresses) end of the Eden. The unit of
allocation is a pointer-sized **cell**. Byte-sized things like strings get
rounded up to a whole number of cells. The minimum size is two cells.

Allocation is very cheap - subtract the size from the pointer and compare it
with the low end of Eden. To keep the cost low, the Eden pointer should live
permanently in a CPU register, if at all possible.

When the allocator is asked to allocate _k_ cells and there's not enough space
left in Eden, a **minor collection** is begun.

```
  Tenured   Old       Reserve            Eden
+----------------------------------------------------------+
|XXXXXXXXX|XXXXXXXXX|                  |XXXXXXXXXXXXXXXXXXX|
+----------------------------------------------------------+
```

### Lifecycle 2 - Minor collection

At the start of a minor collection, the arena will look like this:

Two GC "registers" are defined, `scan` and `store`, both pointing at the
low end of the Reserve space. A minor collection has three phases.

Phase 1 is a depth-first scan from each of the **GC roots**, with the recursion
stopping when it encounters any Lisp value. If the value resides in the Tenured
or Old spaces, it is left alone. If the value resides in Eden, it is copied to
`store`, then `store` is moved to the cell after the copied value. The original
location in Eden is updated as well - the first cell (the header) is replaced
with 0, as a marker for future scans. The second cell is updated to hold the
new location; this is referred to as the **fowarding pointer**.

Some extra notes about phase 1:
- If the Eden value found has 0 in its first cell, it's already been forwarded
and can be skipped.
- The GC roots are native structures like threads, not Lisp values.
- To be clear, the scan in phase 1 **does not recurse** into any Lisp values;
  that happens in phase 3.

Phase 2 is the scan of a special GC root: the **escaped set**. This is a native
linked list of pointers to cells _outside_ Eden which at some point contained
pointers _to_ Eden. This list is scanned, and the cells it points to are
checked. If they still contain pointers to Eden (they might have been mutated
again and now point to a value outside Eden), the Eden value is copied and
forwarded just like in phase 1. The escaped set can be discarded after it has
been fully scanned (indicated by a tail pointer of `0`).

Now at the start of phase 3, the region of Reserve bounded by `scan` and
`store` contains 0 or more values lately in Eden and now copied to Reserve.
These values are exactly those from Eden which are reachable from outside
Eden. Phase 3 is a breadth-first scan of all the live objects from Eden,
starting from this set. The Reserve space is itself the BFS queue, managed by
the `scan` and `store` pointers.

The value at `scan` is scanned, and any pointers to values in Eden which are
not already forwarded are copied and forwarded, then the pointer in the value
at `scan` is updated to the new location. Then `scan` is advanced to the next
value, and the search continues. When `scan` catches up to `store`, the
collection is complete.

Note a vital point about this algorithm: it only touches the survivors, not
the garbage! In this model, large amounts of data are allocated but only a tiny
fraction of it survives the next minor collection.

Now that the Old region has grown, the remaining free space is divided evenly
between Reserve and Eden. The allocation pointer is reset to just after Eden,
and the original allocator call that triggered the collection is handled.

```
========================== Before ==========================
  Tenured   Old       Reserve            Eden
+----------------------------------------------------------+
|XXXXXXXXX|XXXXXXXXX|                  |XXXXXXXXXXXXXXXXXXX|
+----------------------------------------------------------+

============== After - "NNN" is newly copied ===============
  Tenured   Old          Reserve         Eden
+----------------------------------------------------------+
|XXXXXXXXX|XXXXXXXXXNNN|                 |                X|
+----------------------------------------------------------+
```


### Lifecycle 3 - Major collection

To ensure there's always enough free space to perform garbage collection, it's
vital that the Old region is less than half the non-Tenured space. If after a
minor collection the space looks like this:
```
  Tenured   Old
+----------------------------------------------------------+
|XXXXXXXXX|XXXXXXXXXXXXXXXXXXXXXXXNNN|                     |
+----------------------------------------------------------+
                                   ^- midpoint
```
then a **major collection** is necessary. A major collection is much more
expensive, since it scans *every* live value, rather than a few roots and the
(probably very few) survivors in Eden. Since a minor collection just finished,
the Eden is empty.

The `NNN` region has just survived a minor collection, so it's all live data.
(More or less - it's possible the escaped set contains pointers in dead Old
values, and then dead Eden values might survive wrongly. This is not a problem
in practice.) It is left where it is, and so is Tenured space.

All the GC roots and every value in Tenured and `NNN` space get scanned, and
all the reachable Old values are copied to the start of the free area just
right of `NNN`, which we may call Old'.

```
  Tenured   Old                       Old'
+----------------------------------------------------------+
|XXXXXXXXX|XXXXXXXXXXXXXXXXXXXXXXXNNN|XXXXXXXXXX           |
+----------------------------------------------------------+
                                   ^- midpoint
```

Then the values in Old' between `scan` and `space` are treated as a queue for a
breadth-first search, just like in a minor collection.

```
  Tenured                             Old'
+----------------------------------------------------------+
|XXXXXXXXX|                       NNN|XXXXXXXXXX           |
+----------------------------------------------------------+
                                   ^- midpoint
```

Finally, when that scan is complete, `NNN` and Old' are copied and forwarded
again, to the area just above Tenured space. This becomes the new, smaller Old
space, the free area is divided between Reserve and Eden, and GC is complete.

```
  Tenured     Old          Reserve          Eden
+----------------------------------------------------------+
|XXXXXXXXXXX|XXXXXXXXXXXX|                |                |
+----------------------------------------------------------+
```

### Lifecycle 4 - Tenuring

Values are promoted to Tenured status after they have been major-collected a
certain number of times. This reduces the time wasted copying probably-immortal
values. However, since Tenured space is adjacent to occupied Old space, they
can't be copied there right away. Instead a total size of newly tenured values
is maintained, and the values are copied to Old' like any other. The GC count
is incremented for every object copied to Old', and checked against the
Tenuring threshold. If it needs to be Tenured, increase the NewTenure size.

In the final phase of the major collection, the Tenured area is expanded to
make space for the newly tenured values, and Old comes after that. The tenuring
threshold is checked again and `NNN`/Old' values are either copied to the
Tenured space or to Old.

Currently, Tenured object live forever. They can be also be collected, though
the scan is very expensive. The Tenured area can be collected after a major
collection, in a similar way to a major collection - copying to Tenured' in
Reserve/Eden, and then back.

That leaves a gap between the new, slightly smaller Tenured space and Old.
On the next major collection, the Old region can be moved left.

Generally speaking, programs don't run long enough for the amount of tenured
garbage to become a problem.

