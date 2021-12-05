(() => {
  // node_modules/solid-js/dist/solid.js
  var sharedConfig = {};
  function setHydrateContext(context) {
    sharedConfig.context = context;
  }
  function nextHydrateContext() {
    return {
      ...sharedConfig.context,
      id: `${sharedConfig.context.id}${sharedConfig.context.count++}.`,
      count: 0
    };
  }
  var equalFn = (a, b) => a === b;
  var $PROXY = Symbol("solid-proxy");
  var signalOptions = {
    equals: equalFn
  };
  var ERROR = null;
  var runEffects = runQueue;
  var NOTPENDING = {};
  var STALE = 1;
  var PENDING = 2;
  var UNOWNED = {
    owned: null,
    cleanups: null,
    context: null,
    owner: null
  };
  var [transPending, setTransPending] = /* @__PURE__ */ createSignal(false);
  var Owner = null;
  var Transition = null;
  var Scheduler = null;
  var Listener = null;
  var Pending = null;
  var Updates = null;
  var Effects = null;
  var ExecCount = 0;
  function createRoot(fn, detachedOwner) {
    detachedOwner && (Owner = detachedOwner);
    const listener = Listener, owner = Owner, root = fn.length === 0 && true ? UNOWNED : {
      owned: null,
      cleanups: null,
      context: null,
      owner
    };
    Owner = root;
    Listener = null;
    let result;
    try {
      runUpdates(() => result = fn(() => cleanNode(root)), true);
    } finally {
      Listener = listener;
      Owner = owner;
    }
    return result;
  }
  function createSignal(value, options) {
    options = options ? Object.assign({}, signalOptions, options) : signalOptions;
    const s = {
      value,
      observers: null,
      observerSlots: null,
      pending: NOTPENDING,
      comparator: options.equals || void 0
    };
    return [readSignal.bind(s), (value2) => {
      if (typeof value2 === "function") {
        if (Transition && Transition.running && Transition.sources.has(s))
          value2 = value2(s.pending !== NOTPENDING ? s.pending : s.tValue);
        else
          value2 = value2(s.pending !== NOTPENDING ? s.pending : s.value);
      }
      return writeSignal(s, value2);
    }];
  }
  function createComputed(fn, value, options) {
    const c = createComputation(fn, value, true, STALE);
    if (Scheduler && Transition && Transition.running)
      Updates.push(c);
    else
      updateComputation(c);
  }
  function createRenderEffect(fn, value, options) {
    const c = createComputation(fn, value, false, STALE);
    if (Scheduler && Transition && Transition.running)
      Updates.push(c);
    else
      updateComputation(c);
  }
  function createMemo(fn, value, options) {
    options = options ? Object.assign({}, signalOptions, options) : signalOptions;
    const c = createComputation(fn, value, true, 0);
    c.pending = NOTPENDING;
    c.observers = null;
    c.observerSlots = null;
    c.comparator = options.equals || void 0;
    if (Scheduler && Transition && Transition.running) {
      c.tState = STALE;
      Updates.push(c);
    } else
      updateComputation(c);
    return readSignal.bind(c);
  }
  function batch(fn) {
    if (Pending)
      return fn();
    let result;
    const q = Pending = [];
    try {
      result = fn();
    } finally {
      Pending = null;
    }
    runUpdates(() => {
      for (let i = 0; i < q.length; i += 1) {
        const data = q[i];
        if (data.pending !== NOTPENDING) {
          const pending = data.pending;
          data.pending = NOTPENDING;
          writeSignal(data, pending);
        }
      }
    }, false);
    return result;
  }
  function untrack(fn) {
    let result, listener = Listener;
    Listener = null;
    result = fn();
    Listener = listener;
    return result;
  }
  function onCleanup(fn) {
    if (Owner === null)
      ;
    else if (Owner.cleanups === null)
      Owner.cleanups = [fn];
    else
      Owner.cleanups.push(fn);
    return fn;
  }
  function createContext(defaultValue) {
    const id = Symbol("context");
    return {
      id,
      Provider: createProvider(id),
      defaultValue
    };
  }
  function children(fn) {
    const children2 = createMemo(fn);
    return createMemo(() => resolveChildren(children2()));
  }
  function readSignal() {
    const runningTransition = Transition && Transition.running;
    if (this.sources && (!runningTransition && this.state || runningTransition && this.tState)) {
      const updates = Updates;
      Updates = null;
      !runningTransition && this.state === STALE || runningTransition && this.tState === STALE ? updateComputation(this) : lookDownstream(this);
      Updates = updates;
    }
    if (Listener) {
      const sSlot = this.observers ? this.observers.length : 0;
      if (!Listener.sources) {
        Listener.sources = [this];
        Listener.sourceSlots = [sSlot];
      } else {
        Listener.sources.push(this);
        Listener.sourceSlots.push(sSlot);
      }
      if (!this.observers) {
        this.observers = [Listener];
        this.observerSlots = [Listener.sources.length - 1];
      } else {
        this.observers.push(Listener);
        this.observerSlots.push(Listener.sources.length - 1);
      }
    }
    if (runningTransition && Transition.sources.has(this))
      return this.tValue;
    return this.value;
  }
  function writeSignal(node, value, isComp) {
    if (node.comparator) {
      if (Transition && Transition.running && Transition.sources.has(node)) {
        if (node.comparator(node.tValue, value))
          return value;
      } else if (node.comparator(node.value, value))
        return value;
    }
    if (Pending) {
      if (node.pending === NOTPENDING)
        Pending.push(node);
      node.pending = value;
      return value;
    }
    let TransitionRunning = false;
    if (Transition) {
      TransitionRunning = Transition.running;
      if (TransitionRunning || !isComp && Transition.sources.has(node)) {
        Transition.sources.add(node);
        node.tValue = value;
      }
      if (!TransitionRunning)
        node.value = value;
    } else
      node.value = value;
    if (node.observers && node.observers.length) {
      runUpdates(() => {
        for (let i = 0; i < node.observers.length; i += 1) {
          const o = node.observers[i];
          if (TransitionRunning && Transition.disposed.has(o))
            continue;
          if (o.pure)
            Updates.push(o);
          else
            Effects.push(o);
          if (o.observers && (TransitionRunning && !o.tState || !TransitionRunning && !o.state))
            markUpstream(o);
          if (TransitionRunning)
            o.tState = STALE;
          else
            o.state = STALE;
        }
        if (Updates.length > 1e6) {
          Updates = [];
          if (false)
            ;
          throw new Error();
        }
      }, false);
    }
    return value;
  }
  function updateComputation(node) {
    if (!node.fn)
      return;
    cleanNode(node);
    const owner = Owner, listener = Listener, time = ExecCount;
    Listener = Owner = node;
    runComputation(node, Transition && Transition.running && Transition.sources.has(node) ? node.tValue : node.value, time);
    if (Transition && !Transition.running && Transition.sources.has(node)) {
      queueMicrotask(() => {
        runUpdates(() => {
          Transition && (Transition.running = true);
          runComputation(node, node.tValue, time);
        }, false);
      });
    }
    Listener = listener;
    Owner = owner;
  }
  function runComputation(node, value, time) {
    let nextValue;
    try {
      nextValue = node.fn(value);
    } catch (err) {
      handleError(err);
    }
    if (!node.updatedAt || node.updatedAt <= time) {
      if (node.observers && node.observers.length) {
        writeSignal(node, nextValue, true);
      } else if (Transition && Transition.running && node.pure) {
        Transition.sources.add(node);
        node.tValue = nextValue;
      } else
        node.value = nextValue;
      node.updatedAt = time;
    }
  }
  function createComputation(fn, init, pure, state = STALE, options) {
    const c = {
      fn,
      state,
      updatedAt: null,
      owned: null,
      sources: null,
      sourceSlots: null,
      cleanups: null,
      value: init,
      owner: Owner,
      context: null,
      pure
    };
    if (Transition && Transition.running) {
      c.state = 0;
      c.tState = state;
    }
    if (Owner === null)
      ;
    else if (Owner !== UNOWNED) {
      if (Transition && Transition.running && Owner.pure) {
        if (!Owner.tOwned)
          Owner.tOwned = [c];
        else
          Owner.tOwned.push(c);
      } else {
        if (!Owner.owned)
          Owner.owned = [c];
        else
          Owner.owned.push(c);
      }
    }
    return c;
  }
  function runTop(node) {
    const runningTransition = Transition && Transition.running;
    if (!runningTransition && node.state !== STALE)
      return node.state = 0;
    if (runningTransition && node.tState !== STALE)
      return node.tState = 0;
    if (node.suspense && untrack(node.suspense.inFallback))
      return node.suspense.effects.push(node);
    const ancestors = [node];
    while ((node = node.owner) && (!node.updatedAt || node.updatedAt < ExecCount)) {
      if (runningTransition && Transition.disposed.has(node))
        return;
      if (!runningTransition && node.state || runningTransition && node.tState)
        ancestors.push(node);
    }
    for (let i = ancestors.length - 1; i >= 0; i--) {
      node = ancestors[i];
      if (runningTransition) {
        let top = node, prev = ancestors[i + 1];
        while ((top = top.owner) && top !== prev) {
          if (Transition.disposed.has(top))
            return;
        }
      }
      if (!runningTransition && node.state === STALE || runningTransition && node.tState === STALE) {
        updateComputation(node);
      } else if (!runningTransition && node.state === PENDING || runningTransition && node.tState === PENDING) {
        const updates = Updates;
        Updates = null;
        lookDownstream(node);
        Updates = updates;
      }
    }
  }
  function runUpdates(fn, init) {
    if (Updates)
      return fn();
    let wait = false;
    if (!init)
      Updates = [];
    if (Effects)
      wait = true;
    else
      Effects = [];
    ExecCount++;
    try {
      fn();
    } catch (err) {
      handleError(err);
    } finally {
      completeUpdates(wait);
    }
  }
  function completeUpdates(wait) {
    if (Updates) {
      if (Scheduler && Transition && Transition.running)
        scheduleQueue(Updates);
      else
        runQueue(Updates);
      Updates = null;
    }
    if (wait)
      return;
    let cbs;
    if (Transition && Transition.running) {
      if (Transition.promises.size || Transition.queue.size) {
        Transition.running = false;
        Transition.effects.push.apply(Transition.effects, Effects);
        Effects = null;
        setTransPending(true);
        return;
      }
      const sources = Transition.sources;
      cbs = Transition.cb;
      Effects.forEach((e) => {
        "tState" in e && (e.state = e.tState);
        delete e.tState;
      });
      Transition = null;
      batch(() => {
        sources.forEach((v) => {
          v.value = v.tValue;
          if (v.owned) {
            for (let i = 0, len = v.owned.length; i < len; i++)
              cleanNode(v.owned[i]);
          }
          if (v.tOwned)
            v.owned = v.tOwned;
          delete v.tValue;
          delete v.tOwned;
          v.tState = 0;
        });
        setTransPending(false);
      });
    }
    if (Effects.length)
      batch(() => {
        runEffects(Effects);
        Effects = null;
      });
    else {
      Effects = null;
    }
    if (cbs)
      cbs.forEach((cb) => cb());
  }
  function runQueue(queue) {
    for (let i = 0; i < queue.length; i++)
      runTop(queue[i]);
  }
  function scheduleQueue(queue) {
    for (let i = 0; i < queue.length; i++) {
      const item = queue[i];
      const tasks = Transition.queue;
      if (!tasks.has(item)) {
        tasks.add(item);
        Scheduler(() => {
          tasks.delete(item);
          runUpdates(() => {
            Transition.running = true;
            runTop(item);
            if (!tasks.size) {
              Effects.push.apply(Effects, Transition.effects);
              Transition.effects = [];
            }
          }, false);
          Transition && (Transition.running = false);
        });
      }
    }
  }
  function lookDownstream(node) {
    node.state = 0;
    const runningTransition = Transition && Transition.running;
    for (let i = 0; i < node.sources.length; i += 1) {
      const source = node.sources[i];
      if (source.sources) {
        if (!runningTransition && source.state === STALE || runningTransition && source.tState === STALE)
          runTop(source);
        else if (!runningTransition && source.state === PENDING || runningTransition && source.tState === PENDING)
          lookDownstream(source);
      }
    }
  }
  function markUpstream(node) {
    const runningTransition = Transition && Transition.running;
    for (let i = 0; i < node.observers.length; i += 1) {
      const o = node.observers[i];
      if (!runningTransition && !o.state || runningTransition && !o.tState) {
        if (runningTransition)
          o.tState = PENDING;
        else
          o.state = PENDING;
        if (o.pure)
          Updates.push(o);
        else
          Effects.push(o);
        o.observers && markUpstream(o);
      }
    }
  }
  function cleanNode(node) {
    let i;
    if (node.sources) {
      while (node.sources.length) {
        const source = node.sources.pop(), index = node.sourceSlots.pop(), obs = source.observers;
        if (obs && obs.length) {
          const n = obs.pop(), s = source.observerSlots.pop();
          if (index < obs.length) {
            n.sourceSlots[s] = index;
            obs[index] = n;
            source.observerSlots[index] = s;
          }
        }
      }
    }
    if (Transition && Transition.running && node.pure) {
      if (node.tOwned) {
        for (i = 0; i < node.tOwned.length; i++)
          cleanNode(node.tOwned[i]);
        delete node.tOwned;
      }
      reset(node, true);
    } else if (node.owned) {
      for (i = 0; i < node.owned.length; i++)
        cleanNode(node.owned[i]);
      node.owned = null;
    }
    if (node.cleanups) {
      for (i = 0; i < node.cleanups.length; i++)
        node.cleanups[i]();
      node.cleanups = null;
    }
    if (Transition && Transition.running)
      node.tState = 0;
    else
      node.state = 0;
    node.context = null;
  }
  function reset(node, top) {
    if (!top) {
      node.tState = 0;
      Transition.disposed.add(node);
    }
    if (node.owned) {
      for (let i = 0; i < node.owned.length; i++)
        reset(node.owned[i]);
    }
  }
  function handleError(err) {
    const fns = ERROR && lookup(Owner, ERROR);
    if (!fns)
      throw err;
    fns.forEach((f) => f(err));
  }
  function lookup(owner, key) {
    return owner && (owner.context && owner.context[key] || owner.owner && lookup(owner.owner, key));
  }
  function resolveChildren(children2) {
    if (typeof children2 === "function" && !children2.length)
      return resolveChildren(children2());
    if (Array.isArray(children2)) {
      const results = [];
      for (let i = 0; i < children2.length; i++) {
        const result = resolveChildren(children2[i]);
        Array.isArray(result) ? results.push.apply(results, result) : results.push(result);
      }
      return results;
    }
    return children2;
  }
  function createProvider(id) {
    return function provider(props) {
      let res;
      createComputed(() => res = untrack(() => {
        Owner.context = {
          [id]: props.value
        };
        return children(() => props.children);
      }));
      return res;
    };
  }
  var FALLBACK = Symbol("fallback");
  function dispose(d) {
    for (let i = 0; i < d.length; i++)
      d[i]();
  }
  function mapArray(list, mapFn, options = {}) {
    let items = [], mapped = [], disposers = [], len = 0, indexes = mapFn.length > 1 ? [] : null;
    onCleanup(() => dispose(disposers));
    return () => {
      let newItems = list() || [], i, j;
      return untrack(() => {
        let newLen = newItems.length, newIndices, newIndicesNext, temp, tempdisposers, tempIndexes, start, end, newEnd, item;
        if (newLen === 0) {
          if (len !== 0) {
            dispose(disposers);
            disposers = [];
            items = [];
            mapped = [];
            len = 0;
            indexes && (indexes = []);
          }
          if (options.fallback) {
            items = [FALLBACK];
            mapped[0] = createRoot((disposer) => {
              disposers[0] = disposer;
              return options.fallback();
            });
            len = 1;
          }
        } else if (len === 0) {
          mapped = new Array(newLen);
          for (j = 0; j < newLen; j++) {
            items[j] = newItems[j];
            mapped[j] = createRoot(mapper);
          }
          len = newLen;
        } else {
          temp = new Array(newLen);
          tempdisposers = new Array(newLen);
          indexes && (tempIndexes = new Array(newLen));
          for (start = 0, end = Math.min(len, newLen); start < end && items[start] === newItems[start]; start++)
            ;
          for (end = len - 1, newEnd = newLen - 1; end >= start && newEnd >= start && items[end] === newItems[newEnd]; end--, newEnd--) {
            temp[newEnd] = mapped[end];
            tempdisposers[newEnd] = disposers[end];
            indexes && (tempIndexes[newEnd] = indexes[end]);
          }
          newIndices = /* @__PURE__ */ new Map();
          newIndicesNext = new Array(newEnd + 1);
          for (j = newEnd; j >= start; j--) {
            item = newItems[j];
            i = newIndices.get(item);
            newIndicesNext[j] = i === void 0 ? -1 : i;
            newIndices.set(item, j);
          }
          for (i = start; i <= end; i++) {
            item = items[i];
            j = newIndices.get(item);
            if (j !== void 0 && j !== -1) {
              temp[j] = mapped[i];
              tempdisposers[j] = disposers[i];
              indexes && (tempIndexes[j] = indexes[i]);
              j = newIndicesNext[j];
              newIndices.set(item, j);
            } else
              disposers[i]();
          }
          for (j = start; j < newLen; j++) {
            if (j in temp) {
              mapped[j] = temp[j];
              disposers[j] = tempdisposers[j];
              if (indexes) {
                indexes[j] = tempIndexes[j];
                indexes[j](j);
              }
            } else
              mapped[j] = createRoot(mapper);
          }
          mapped = mapped.slice(0, len = newLen);
          items = newItems.slice(0);
        }
        return mapped;
      });
      function mapper(disposer) {
        disposers[j] = disposer;
        if (indexes) {
          const [s, set] = createSignal(j);
          indexes[j] = set;
          return mapFn(newItems[j], s);
        }
        return mapFn(newItems[j]);
      }
    };
  }
  function createComponent(Comp, props) {
    if (sharedConfig.context) {
      const c = sharedConfig.context;
      setHydrateContext(nextHydrateContext());
      const r = untrack(() => Comp(props));
      setHydrateContext(c);
      return r;
    }
    return untrack(() => Comp(props));
  }
  function For(props) {
    const fallback = "fallback" in props && {
      fallback: () => props.fallback
    };
    return createMemo(mapArray(() => props.each, props.children, fallback ? fallback : void 0));
  }
  function Show(props) {
    let strictEqual = false;
    const condition = createMemo(() => props.when, void 0, {
      equals: (a, b) => strictEqual ? a === b : !a === !b
    });
    return createMemo(() => {
      const c = condition();
      if (c) {
        const child = props.children;
        return (strictEqual = typeof child === "function" && child.length > 0) ? untrack(() => child(c)) : child;
      }
      return props.fallback;
    });
  }
  var SuspenseListContext = createContext();

  // node_modules/solid-js/web/dist/web.js
  var booleans = ["allowfullscreen", "async", "autofocus", "autoplay", "checked", "controls", "default", "disabled", "formnovalidate", "hidden", "indeterminate", "ismap", "loop", "multiple", "muted", "nomodule", "novalidate", "open", "playsinline", "readonly", "required", "reversed", "seamless", "selected"];
  var Properties = /* @__PURE__ */ new Set(["className", "value", "readOnly", "formNoValidate", "isMap", "noModule", "playsInline", ...booleans]);
  function reconcileArrays(parentNode, a, b) {
    let bLength = b.length, aEnd = a.length, bEnd = bLength, aStart = 0, bStart = 0, after = a[aEnd - 1].nextSibling, map = null;
    while (aStart < aEnd || bStart < bEnd) {
      if (a[aStart] === b[bStart]) {
        aStart++;
        bStart++;
        continue;
      }
      while (a[aEnd - 1] === b[bEnd - 1]) {
        aEnd--;
        bEnd--;
      }
      if (aEnd === aStart) {
        const node = bEnd < bLength ? bStart ? b[bStart - 1].nextSibling : b[bEnd - bStart] : after;
        while (bStart < bEnd)
          parentNode.insertBefore(b[bStart++], node);
      } else if (bEnd === bStart) {
        while (aStart < aEnd) {
          if (!map || !map.has(a[aStart]))
            parentNode.removeChild(a[aStart]);
          aStart++;
        }
      } else if (a[aStart] === b[bEnd - 1] && b[bStart] === a[aEnd - 1]) {
        const node = a[--aEnd].nextSibling;
        parentNode.insertBefore(b[bStart++], a[aStart++].nextSibling);
        parentNode.insertBefore(b[--bEnd], node);
        a[aEnd] = b[bEnd];
      } else {
        if (!map) {
          map = /* @__PURE__ */ new Map();
          let i = bStart;
          while (i < bEnd)
            map.set(b[i], i++);
        }
        const index = map.get(a[aStart]);
        if (index != null) {
          if (bStart < index && index < bEnd) {
            let i = aStart, sequence = 1, t;
            while (++i < aEnd && i < bEnd) {
              if ((t = map.get(a[i])) == null || t !== index + sequence)
                break;
              sequence++;
            }
            if (sequence > index - bStart) {
              const node = a[aStart];
              while (bStart < index)
                parentNode.insertBefore(b[bStart++], node);
            } else
              parentNode.replaceChild(b[bStart++], a[aStart++]);
          } else
            aStart++;
        } else
          parentNode.removeChild(a[aStart++]);
      }
    }
  }
  function render(code, element, init) {
    let disposer;
    createRoot((dispose2) => {
      disposer = dispose2;
      insert(element, code(), element.firstChild ? null : void 0, init);
    });
    return () => {
      disposer();
      element.textContent = "";
    };
  }
  function template(html, check, isSVG) {
    const t = document.createElement("template");
    t.innerHTML = html;
    let node = t.content.firstChild;
    if (isSVG)
      node = node.firstChild;
    return node;
  }
  function insert(parent, accessor, marker, initial) {
    if (marker !== void 0 && !initial)
      initial = [];
    if (typeof accessor !== "function")
      return insertExpression(parent, accessor, initial, marker);
    createRenderEffect((current) => insertExpression(parent, accessor(), current, marker), initial);
  }
  function insertExpression(parent, value, current, marker, unwrapArray) {
    while (typeof current === "function")
      current = current();
    if (value === current)
      return current;
    const t = typeof value, multi = marker !== void 0;
    parent = multi && current[0] && current[0].parentNode || parent;
    if (t === "string" || t === "number") {
      if (t === "number")
        value = value.toString();
      if (multi) {
        let node = current[0];
        if (node && node.nodeType === 3) {
          node.data = value;
        } else
          node = document.createTextNode(value);
        current = cleanChildren(parent, current, marker, node);
      } else {
        if (current !== "" && typeof current === "string") {
          current = parent.firstChild.data = value;
        } else
          current = parent.textContent = value;
      }
    } else if (value == null || t === "boolean") {
      if (sharedConfig.context)
        return current;
      current = cleanChildren(parent, current, marker);
    } else if (t === "function") {
      createRenderEffect(() => {
        let v = value();
        while (typeof v === "function")
          v = v();
        current = insertExpression(parent, v, current, marker);
      });
      return () => current;
    } else if (Array.isArray(value)) {
      const array = [];
      if (normalizeIncomingArray(array, value, unwrapArray)) {
        createRenderEffect(() => current = insertExpression(parent, array, current, marker, true));
        return () => current;
      }
      if (sharedConfig.context && current && current.length)
        return current;
      if (array.length === 0) {
        current = cleanChildren(parent, current, marker);
        if (multi)
          return current;
      } else {
        if (Array.isArray(current)) {
          if (current.length === 0) {
            appendNodes(parent, array, marker);
          } else
            reconcileArrays(parent, current, array);
        } else if (current == null || current === "") {
          appendNodes(parent, array);
        } else {
          reconcileArrays(parent, multi && current || [parent.firstChild], array);
        }
      }
      current = array;
    } else if (value instanceof Node) {
      if (Array.isArray(current)) {
        if (multi)
          return current = cleanChildren(parent, current, marker, value);
        cleanChildren(parent, current, null, value);
      } else if (current == null || current === "" || !parent.firstChild) {
        parent.appendChild(value);
      } else
        parent.replaceChild(value, parent.firstChild);
      current = value;
    } else
      ;
    return current;
  }
  function normalizeIncomingArray(normalized, array, unwrap) {
    let dynamic = false;
    for (let i = 0, len = array.length; i < len; i++) {
      let item = array[i], t;
      if (item instanceof Node) {
        normalized.push(item);
      } else if (item == null || item === true || item === false)
        ;
      else if (Array.isArray(item)) {
        dynamic = normalizeIncomingArray(normalized, item) || dynamic;
      } else if ((t = typeof item) === "string") {
        normalized.push(document.createTextNode(item));
      } else if (t === "function") {
        if (unwrap) {
          while (typeof item === "function")
            item = item();
          dynamic = normalizeIncomingArray(normalized, Array.isArray(item) ? item : [item]) || dynamic;
        } else {
          normalized.push(item);
          dynamic = true;
        }
      } else
        normalized.push(document.createTextNode(item.toString()));
    }
    return dynamic;
  }
  function appendNodes(parent, array, marker) {
    for (let i = 0, len = array.length; i < len; i++)
      parent.insertBefore(array[i], marker);
  }
  function cleanChildren(parent, current, marker, replacement) {
    if (marker === void 0)
      return parent.textContent = "";
    const node = replacement || document.createTextNode("");
    if (current.length) {
      let inserted = false;
      for (let i = current.length - 1; i >= 0; i--) {
        const el = current[i];
        if (node !== el) {
          const isParent = el.parentNode === parent;
          if (!inserted && !i)
            isParent ? parent.replaceChild(node, el) : parent.insertBefore(node, marker);
          else
            isParent && parent.removeChild(el);
        } else
          inserted = true;
      }
    } else
      parent.insertBefore(node, marker);
    return [node];
  }

  // static/js/bar.jsx
  var _tmpl$ = template(`<div class="container"><div class="row"><div class="col-sm-10 col-lg-6 offset-lg-3 offset-sm-1 gy-1"><div class="input-group"><button class="btn btn-success">+</button><div class="input-group-text"><input class="form-check-input mt-0" type="checkbox"></div><div class="input-group-text"><input class="form-check-input mt-0" type="checkbox"></div><input class="form-control form-control-sm search-box" type="text"><button class="btn btn-secondary">X</button></div></div></div></div>`, 19);
  function bar_default() {
    return _tmpl$.cloneNode(true);
  }

  // static/js/item.tsx
  var _tmpl$2 = template(`<li class="list-group-item list-group-item-action"><input type="checkbox" class="form-check-input float-start me-1 h5"><div class="overflow-auto item-name"><span></span></div><div class="item-actions"><button class="btn btn-link link-secondary">\u270F\uFE0F</button></div></li>`, 11);
  function item_default(props) {
    return (() => {
      const _el$ = _tmpl$2.cloneNode(true), _el$2 = _el$.firstChild, _el$3 = _el$2.nextSibling, _el$4 = _el$3.firstChild;
      insert(_el$4, () => props.item.name);
      createRenderEffect((_p$) => {
        const _v$ = props.item.is_checked, _v$2 = props.item.is_checked;
        _v$ !== _p$._v$ && _el$.classList.toggle("list-group-item-light", _p$._v$ = _v$);
        _v$2 !== _p$._v$2 && (_el$2.checked = _p$._v$2 = _v$2);
        return _p$;
      }, {
        _v$: void 0,
        _v$2: void 0
      });
      return _el$;
    })();
  }

  // static/js/items.jsx
  var _tmpl$3 = template(`<div class="mt-4"><ul class="list-group shadow-sm mt-1"></ul></div>`, 4);
  var _tmpl$22 = template(`<span></span>`, 2);
  function items_default() {
    const [items, setItems] = createSignal([{
      "id": 1,
      "category": "test",
      "name": "asdf1",
      "is_checked": false
    }, {
      "id": 1,
      "category": "test",
      "name": "asdf2",
      "is_checked": true
    }, {
      "id": 1,
      "category": "test2",
      "name": "asdf3",
      "is_checked": false
    }]);
    const showCategory = function(i) {
      if (i == 0) {
        return true;
      }
      if (i > 0 && i < items().length) {
        if (items()[i].category !== items()[i - 1].category) {
          return true;
        }
      }
      return false;
    };
    return (() => {
      const _el$ = _tmpl$3.cloneNode(true), _el$2 = _el$.firstChild;
      insert(_el$2, createComponent(For, {
        get each() {
          return items();
        },
        children: (item, i) => [createComponent(Show, {
          get when() {
            return showCategory(i());
          },
          get children() {
            const _el$3 = _tmpl$22.cloneNode(true);
            insert(_el$3, () => item.category);
            return _el$3;
          }
        }), createComponent(item_default, {
          item
        })]
      }));
      return _el$;
    })();
  }

  // static/js/app.tsx
  var _tmpl$4 = template(`<div class="container"><div class="row"><div class="col-sm-10 col-lg-6 offset-lg-3 offset-sm-1 gy-4"><div class="position-fixed fixed-top bg-white"></div></div></div></div>`, 8);
  function App() {
    return (() => {
      const _el$ = _tmpl$4.cloneNode(true), _el$2 = _el$.firstChild, _el$3 = _el$2.firstChild, _el$4 = _el$3.firstChild;
      insert(_el$4, createComponent(bar_default, {}));
      insert(_el$3, createComponent(items_default, {}), null);
      return _el$;
    })();
  }
  render(() => createComponent(App, {}), document.getElementById("app"));
})();
