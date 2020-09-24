const app = new Vue({
  el: "#app",
  data: { url: "", shortlink: "", message:"" },
  computed: {},
  methods: {
    resetFields: function () {
      this.url = "";
    },
    shortenUrl: function () {
      fetch(`/`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: this.url }),
      })
        .then((resp) => resp.json())
        .then((result) => {
          if (result.hasOwnProperty("ShortLink")) {
            this.shortlink = result["ShortLink"];
            this.message = "Your simple URL is generated! You can copy and paste to your browser. Or share with your contacts. "
          }
        })
        .catch((error) => {
          console.log(error);
        });
    },
  },
});
