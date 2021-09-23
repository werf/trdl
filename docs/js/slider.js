$(document).ready(function(){
  const sel = $('.slider__nav');
  const nav = $('.slider__navigation');

  $('.slider__wrap').slick({
    autoplay: false,
    draggable: false,
    infinite: true,
    dots: true,
    speed: 0,
    dotsClass: 'slider__dots',
    appendArrows: sel,
    appendDots: sel,
    prevArrow: `<button type="button" class="slider__prev"><svg class="slider__arrow"><use xlink:href="/images/icons/sprite.svg#arrow-left"></use></svg></button>`,
    nextArrow: `<button type="button" class="slider__next"><svg class="slider__arrow"><use xlink:href="/images/icons/sprite.svg#arrow-left"></use></svg></button>`
  });

  $('.slider__wrapper').slick({
    autoplay: false,
    draggable: false,
    infinite: true,
    dots: true,
    speed: 0,
    dotsClass: 'slider__dots',
    appendArrows: nav,
    appendDots: nav,
    prevArrow: `<button type="button" class="slider__prev"><svg class="slider__arrow"><use xlink:href="/images/icons/sprite.svg#arrow-left"></use></svg></button>`,
    nextArrow: `<button type="button" class="slider__next"><svg class="slider__arrow"><use xlink:href="/images/icons/sprite.svg#arrow-left"></use></svg></button>`
  });
});