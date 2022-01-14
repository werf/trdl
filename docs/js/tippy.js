window.addEventListener('DOMContentLoaded', ()=> {
  tippy('.footnote', {
    interactive: true,
    interactiveDebounce: 75,
    allowHTML: true,
    content(reference) {
      const title = reference.getAttribute('title');
      reference.removeAttribute('title');
      return title;
    },
  });

  // tippy('#vault__en', {
  //   interactive: true,
  //   interactiveDebounce: 75,
  //   allowHTML: true,
  //   content: "HashiCorp's Vault is a secret management tool. In trdl, we use a custom Vault plugin tailored for secure package delivery. <a href='https://www.hashicorp.com/products/vault'>Learn more about Vault</a>."
  // });

  tippy('.required', {
    content(reference) {
      const title = reference.getAttribute('title');
      reference.removeAttribute('title');
      return title;
    },
  });
});
