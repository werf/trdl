window.addEventListener('DOMContentLoaded', ()=> {
  tippy('#your-program__ru', {
    content: "Программой может быть что угодно: бинарный файл, shell-скрипт и даже Ansible-плейбук"
  });

  tippy('#your-program__en', {
    content: 'In this case, "application" refers to any form of programming code, e.g., a binary file, a shell script, and even an Ansible playbook.'
  });

  tippy('#tuf-repo__ru', {
    interactive: true,
    interactiveDebounce: 75,
    allowHTML: true,
    content: "TUF (The Update Framework) — фреймворк, который применяется для защиты систем обновления ПО. TUF-репозиторий — любое хранилище (например, S3), для работы с которым используются инструменты безопасности TUF. Подробнее о TUF: <a href='https://theupdateframework.io/'>https://theupdateframework.io/</a>"
  });

  tippy('#tuf-repo__en', {
    interactive: true,
    interactiveDebounce: 75,
    allowHTML: true,
    content: "TUF (The Update Framework) is a framework for securing software update systems. TUF repository is any repository with your software (e.g., S3) that uses TUF security tools. Learn more about TUF: <a href='https://theupdateframework.io/'>https://theupdateframework.io/</a>"
  });
});
