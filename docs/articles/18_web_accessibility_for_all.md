# 18. One for Accessibility, All for Accessibility

When building frontend applications, I've always found it more meaningful to design for one person,
not for "scale." But who is this one person? Are they able to fully experience what I've built?
Where are they from? Do they have a disability? Am I communicating in a language they understand?
Can they access my assets and media, even with network limitations? These are the questions that
shape my approach to accessibility.

## 1. What Is Web Accessibility?

Web accessibility means making your applications usable for people with disabilities—but it's so
much more. True accessibility is about empathy: creating digital experiences that everyone can
enjoy, regardless of ability, context, or circumstance.

People may face a variety of challenges when using the web, including:

- **Mobility and physical disabilities**
- **Cognitive and neurological differences**
- **Visual impairments**
- **Hearing impairments**

Without a solid understanding of accessibility principles and semantic HTML, it's easy for
JavaScript and CSS to inadvertently create barriers. Overly complex interfaces, "div soup," and
non-semantic markup can make your site unusable for many.

## 2. How People Access the Web

There are many ways people interact with the web, including:

- **Keyboard only navigation**
- **Head wands and mouth sticks**
- **Single-switch (one-button) devices**
- **Screen readers** (software that converts text to speech)

Each of these methods requires thoughtful design and development to ensure a smooth experience.

## 3. The Curb Cut Effect

The "curb cut effect" describes how accessibility features designed for people with disabilities end
up benefiting everyone. Curb cuts in sidewalks help not just wheelchair users, but also parents with
strollers, travelers with luggage, and delivery workers. Similarly, accessible websites are easier
to use for everyone—including people on mobile devices, with slow connections, or in noisy
environments.

Accessible design also improves SEO: semantic HTML helps search engines understand your content, and
features like alt text make your site more discoverable and usable.

## 4. Accessibility Standards

The [Web Content Accessibility Guidelines (WCAG)](https://www.w3.org/WAI/standards-guidelines/wcag/)
are the gold standard for web accessibility. They provide clear, actionable criteria for making your
site accessible. [WebAIM](https://webaim.org/) offers excellent checklists and resources to help you
meet these standards.

## 5. Screen Readers

Screen readers, such as [NVDA](https://www.nvaccess.org/) and
[JAWS](https://www.freedomscientific.com/products/software/jaws/), convert digital text into
synthesized speech. To support them:

- Use descriptive [alt text](https://webaim.org/techniques/alttext/) for images.
- Provide captions and transcripts for audio and video.
- Use semantic HTML so screen readers can interpret your content structure.
- AI is making screen readers smarter, but clear markup is still essential.

## 6. Accessible HTML

- **Use semantic HTML**: Avoid "div soup." Elements like `<header>`, `<nav>`, `<main>`, `<aside>`,
  and `<footer>` provide structure for both users and search engines.
- **Only one `<h1>` per page**: This helps with both accessibility and SEO.
- **Label form fields properly**: Use `<label>` elements, not `<p>` tags, and ensure each label is
  associated with its input.
- **ARIA attributes**: Use
  [ARIA labels](https://developer.mozilla.org/en-US/docs/Web/Accessibility/ARIA/Attributes/aria-label)
  and roles to enhance accessibility, but don't use them as a substitute for semantic HTML.
- **Tab index**: Manage keyboard navigation with `tabindex`.
- **Keyboard events**: Use `onKeyUp` and similar events to support keyboard users.

## 7. ARIA: Accessible Rich Internet Applications

[ARIA](https://www.w3.org/WAI/standards-guidelines/aria/) helps make dynamic content and custom UI
components accessible. Use ARIA roles, states, and properties to describe elements to assistive
technologies. For example, use `aria-checked` for custom checkboxes, and `aria-describedby` to
provide additional context.

**Note:** Overusing ARIA can cause confusion. Prefer native HTML elements whenever possible.

## 8. Forms Management

- **Support keyboard-only users**: Provide clear focus indicators and logical tab order.
- **Don't change pages unexpectedly**: Sudden changes can disorient users.
- **Keyboard shortcuts**: Offer shortcuts, but make them discoverable and avoid conflicts.

## 9. Skip Links

Skip links let users jump directly to main content, bypassing navigation. Place a visually hidden
menu at the top of your site that becomes visible on focus.
[Learn more about skip links](https://webaim.org/techniques/skipnav/).

## 10. Tab Navigation

- **Tabbable elements**: Only interactive elements (links, buttons, form fields) should be
  focusable.
- **Arrow keys**: Use for vertical navigation in menus.
- **Focus management**: Use the
  [DOM API's `activeElement`](https://developer.mozilla.org/en-US/docs/Web/API/Document/activeElement)
  to manage focus, especially in modals (focus trapping).

## 11. Visual Considerations

- **Escape closes modals**: Users expect this.
- **Color contrast**: Use tools like the
  [WebAIM Contrast Checker](https://webaim.org/resources/contrastchecker/) to ensure readability.
- **Don't rely on color alone**: Use icons, text, or patterns to convey meaning.
- **Simulate color blindness**: Tools like
  [NoCoffee for Firefox](https://addons.mozilla.org/en-US/firefox/addon/nocoffee/) help you test
  accessibility.

## 12. Language and Markup

- **Set the page language**: Use the `lang` attribute on `<html>`.
- **Validate your markup**: Use the [W3C Markup Validation Service](https://validator.w3.org/).
- **Internationalization**: Use libraries like [FormatJS](https://formatjs.io/) for i18n.

## 13. Motion and Color Scheme Preferences

- **Reduced motion**: Respect users'
  [prefers-reduced-motion](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-reduced-motion)
  settings.
- **Color scheme**: Support
  [prefers-color-scheme](https://developer.mozilla.org/en-US/docs/Web/CSS/@media/prefers-color-scheme)
  for light/dark mode.

## 14. Tooling

- **Linters**:
  - [eslint-plugin-jsx-a11y](https://github.com/jsx-eslint/eslint-plugin-jsx-a11y)
  - [angular codelyzer](https://github.com/mgechev/codelyzer)
  - [eslint-plugin-vuejs-accessibility](https://github.com/vue-a11y/eslint-plugin-vuejs-accessibility)
- **Design Systems**:
  - [Adobe React Spectrum](https://react-spectrum.adobe.com/react-spectrum/)
  - [Google Material Design](https://m3.material.io/)
- **Developer Tools**:
  - [Deque axe DevTools](https://www.deque.com/axe/devtools/)
  - [Google Lighthouse](https://developers.google.com/web/tools/lighthouse)

## 15. Resources

- [Web Content Accessibility Guidelines (WCAG)](https://www.w3.org/WAI/standards-guidelines/wcag/)
- [WebAIM](https://webaim.org/)
- [Microsoft Inclusive Design](https://www.microsoft.com/design/inclusive/)
- [Global Accessibility Awareness Day](https://accessibility.day/)
- [Accessibility in JavaScript Applications](https://www.smashingmagazine.com/2021/03/accessibility-javascript-applications/)
- [Start Building Accessible Web Applications Today](https://web.dev/accessibility/)
- [Accessibility Tips & Tricks](https://www.a11yproject.com/checklist/)

---

**Special Note:**

- Anchors and buttons must have text content.
- Lists should be meaningful and properly structured.

---

**Remember:**  
Building for one is building for all. Accessibility is not just a checklist—it's a mindset that
leads to better, more inclusive, and more successful digital experiences for everyone.
