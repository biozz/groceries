let loc = window.location;
var scheme = "ws";
if (loc.protocol === "https:") {
  scheme = "wss";
}
const clientID = Math.floor(100000 + Math.random() * 900000);
let socket = new WebSocket(`${scheme}://${loc.host}/ws?client_id=${clientID}`);

const app = Vue.createApp({
  data() {
    return {
      loading: true,
      isModalShown: false,
      modalTitle: "",
      modalType: "",
      searchText: "",
      hideCompleted: false,
      isGrouped: true,
      items: [],
      editItemMode: "add",
      editItemUid: null,
      editItemName: "",
      editItemCategory: "",
      editItemError: "",
      suggestedCategories: [],
      token: "",
      rawToken: "",
      namespacePrefix: "",
      namespace: "",
    };
  },
  computed: {
    searchIsEmpty() {
      return this.searchText !== "" && this.filteredItems.length === 0;
    },
    allCompleted() {
      return this.filteredItems.length === this.completedItems.length;
    },
    filteredCategories() {
      if (this.isGrouped) {
        return [...new Set(this.filteredItems.map((i) => i.category))];
      }
      return ["all"];
    },
    filteredItems() {
      let items = this.items.slice();
      if (this.searchText.length > 0) {
        items = items.filter((item) =>
          item.name.toLowerCase().includes(this.searchText.toLowerCase())
        );
        this.hideCompleted = false;
      }
      if (this.hideCompleted) {
        return items.filter((i) => i.state === "open");
      }
      return items;
    },
    completedItems() {
      return this.filteredItems.filter((item) => item.state === "completed");
    },
    isSettingsModal() {
      return this.modalType === "settings";
    },
    hasSuggestions() {
      return this.suggestedCategories.length > 0;
    },
    isEditItemModeUpdate() {
      return this.editItemMode === "update";
    },
    isGlobalNamespace() {
      return this.namespace === "default" && this.namespacePrefix === "g";
    },
  },
  watch: {
    hideCompleted(val) {
      localStorage.setItem("hideCompleted", val);
    },
    isGrouped(val) {
      localStorage.setItem("isGrouped", val);
    },
  },
  methods: {
    saveToken() {
      localStorage.setItem("token", this.rawToken);
      window.location.reload();
    },
    getHeaders() {
      return {
        "X-WS-Client-ID": clientID,
        "X-Auth-Token": this.token,
        "X-Namespace-Prefix": this.namespacePrefix,
        "X-Namespace": this.namespace,
      };
    },
    setNamespace() {
      let nsParts = window.location.hash.substring(2).split("/");
      if (nsParts.length !== 2) {
        this.namespacePrefix = "g";
        this.namespace = "default";
        return
      }
      this.namespacePrefix = nsParts[0];
      this.namespace = nsParts[1];
    },
    clearSearch() {
      this.searchText = "";
      this.hideCompleted = true;
    },
    showCompleted() {
      this.hideCompleted = false;
    },
    showSettingsModal() {
      this.isModalShown = true;
      this.modalType = "settings";
      this.modalTitle = "Настройки";
    },
    showAddModal() {
      this.isModalShown = true;
      this.modalTitle = "Добавить";
      this.editItemMode = "add";
    },
    showEditModal(item) {
      this.isModalShown = true;
      this.modalTitle = "Изменить";
      this.editItemName = item.name;
      this.editItemCategory = item.category;
      this.editItemMode = "update";
      this.editItemUid = item.uid;
    },
    closeModal() {
      this.isModalShown = false;
      this.modalType = "";
      if (this.editItemMode === "edit") {
        this.editItemMode = "add";
      }
      this.editItemUid = null;
      this.editItemName = "";
      this.editItemCategory = "";
      this.suggestedCategories = [];
    },
    itemsByCategory(category, is_checked) {
      let filterCompleted = function (i) {
        if (is_checked && i.state === "completed") {
          return true;
        }
        if (!is_checked && i.state !== "completed") {
          return true;
        }
        return false;
      };
      if (!this.isGrouped) {
        return this.filteredItems.filter(filterCompleted);
      }
      return this.filteredItems
        .filter((i) => i.category === category)
        .filter(filterCompleted);
    },
    suggestCategories() {
      let val = this.editItemCategory;
      if (val.length < 2) {
        return [];
      }
      let allCategories = [...new Set(this.items.map((i) => i.category))];
      this.suggestedCategories = allCategories.filter((c) =>
        c.toLowerCase().includes(val.toLowerCase())
      );
    },
    setCategory(category) {
      this.editItemCategory = category;
      this.suggestedCategories = [];
    },
    toggleHideEmptyCategories() {
      if (!this.hideCompleted) {
        this.hideEmptyCategories = false;
      }
    },
    async toggle(item) {
      if (item.is_checked) {
        item.is_prechecked = true;
        item.timer = setTimeout(async () => {
          await this.toggleRequest(item);
        }, 1500);
      } else {
        if (item.is_prechecked) {
          clearTimeout(item.timer);
          item.is_prechecked = false;
          return;
        }
        await this.toggleRequest(item);
      }
    },
    async toggleRequest(item) {
      let res = await fetch(`/items/toggle?uid=${item.uid}`, {
        headers: this.getHeaders(),
      });
      let data = await res.json();
      item.is_checked = data.is_checked;
      if (item.is_checked) {
        item.state = "completed";
      } else {
        item.state = "open";
      }
      item.is_prechecked = false;
    },
    async removeItem() {
      let res = await fetch(`/items/delete?uid=${this.editItemUid}`, {
        headers: this.getHeaders(),
      });
      if (!res.ok) {
        this.editItemError = `${res.status} ${res.statusText}`;
        return;
      }
      let idx = this.items.findIndex((i) => i.uid === this.editItemUid);
      this.items.splice(idx, 1);
      this.closeModal();
    },
    async addItem() {
      let res = await fetch(
        `/items/add?name=${this.editItemName}&category=${this.editItemCategory}`,
        { headers: this.getHeaders() }
      );
      if (!res.ok) {
        this.editItemError = `${res.status} ${res.statusText}`;
        return;
      }
      let data = await res.json();
      this.items.push(Object.assign(data, { state: "open" }));
      this.closeModal();
    },
    async updateItem() {
      let res = await fetch(
        `/items/edit?uid=${this.editItemUid}&name=${this.editItemName}&category=${this.editItemCategory}`,
        { headers: this.getHeaders() }
      );
      if (!res.ok) {
        this.editItemError = `${res.status} ${res.statusText}`;
        return;
      }
      let idx = this.items.findIndex((i) => i.uid === this.editItemUid);
      this.items[idx].name = this.editItemName;
      this.items[idx].category = this.editItemCategory;
      this.closeModal();
    },
    async loadItems() {
      this.items = [];
      let res = await fetch("items/", { headers: this.getHeaders() });
      let rawItems = await res.json();
      for (let item of rawItems) {
        let state = "open";
        if (item.is_checked) {
          state = "completed";
        }
        this.items.push({ ...item, state: state });
      }
    },
  },
  async mounted() {
    let urlSearchParams = new URLSearchParams(window.location.search);
    let params = Object.fromEntries(urlSearchParams.entries());

    if (params.token) {
      localStorage.setItem("token", params.token);
      window.location.replace(location.protocol + "//" + location.host);
    }
    this.token = localStorage.getItem("token");
    if (!this.token) {
      return;
    }
    await this.setNamespace();
    window.onhashchange = async (_) => {
      await this.setNamespace();
      await this.loadItems();
    };
    await this.loadItems();
    // Load settings from local storage, if present
    let hideCompletedLocalStorage = localStorage.getItem("hideCompleted");
    if (hideCompletedLocalStorage !== null) {
      this.hideCompleted = JSON.parse(hideCompletedLocalStorage);
    }
    let isGroupedLocalStorage = localStorage.getItem("isGrouped");
    if (isGroupedLocalStorage !== null) {
      this.isGrouped = JSON.parse(isGroupedLocalStorage);
    }

    // websocket events mapping
    socket.addEventListener("open", function (e) {
      console.log(e);
    });
    socket.addEventListener("message", (raw) => {
      let event = JSON.parse(raw.data);
      let idx = null;
      if (event.data.namespace_prefix !== this.namespacePrefix) {
        return
      }
      if (event.data.namespace !== this.namespace) {
        return
      }
      switch (event.type) {
        case "toggle":
          idx = this.items.findIndex((i) => i.uid === event.data.uid);
          let state = "open";
          if (event.data.is_checked) {
            state = "completed";
          }
          this.items[idx].is_checked = event.data.is_checked;
          this.items[idx].state = state;
          break;
        case "edit":
          idx = this.items.findIndex((i) => i.uid === event.data.uid);
          this.items[idx].name = event.data.name;
          this.items[idx].category = event.data.category;
          break;
        case "delete":
          idx = this.items.findIndex((i) => i.uid === event.data.uid);
          this.items.splice(idx, 1);
          break;
        case "add":
          this.items.push(Object.assign(event.data, { state: "open" }));
      }
    });
    this.loading = false;
  },
});

app.mount("#app");
