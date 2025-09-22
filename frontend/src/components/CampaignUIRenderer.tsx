import React from 'react';
import type { CampaignMetadata } from '../store/types/campaign';

// Dynamic UI Renderer Component
interface DynamicUIRendererProps {
  components: Record<string, any>;
  content: Record<string, any>;
  theme: Record<string, any>;
  campaignTitle: string;
  campaignDescription: string;
}

const DynamicUIRenderer: React.FC<DynamicUIRendererProps> = ({
  components,
  content,
  theme,
  campaignTitle,
  campaignDescription
}) => {
  console.log('[DynamicUIRenderer] Rendering components:', {
    componentCount: Object.keys(components).length,
    components: Object.keys(components),
    contentKeys: Object.keys(content),
    themeKeys: Object.keys(theme)
  });

  // Convert CSS properties to React style objects
  const convertPropsToStyles = (props: Record<string, any>): React.CSSProperties => {
    const styles: React.CSSProperties = {};

    Object.entries(props).forEach(([key, value]) => {
      // Handle special cases
      if (key === 'templateColumns') {
        styles.gridTemplateColumns = value;
      } else if (key === 'gap') {
        styles.gap = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'p') {
        styles.padding = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'm') {
        styles.margin = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'w') {
        styles.width = value === 'full' ? '100%' : value;
      } else if (key === 'h') {
        styles.height = value === 'full' ? '100%' : value;
      } else if (key === 'maxW') {
        styles.maxWidth = value;
      } else if (key === 'mx') {
        styles.marginLeft = 'auto';
        styles.marginRight = 'auto';
      } else if (key === 'my') {
        styles.marginTop = typeof value === 'number' ? `${value * 4}px` : value;
        styles.marginBottom = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'px') {
        styles.paddingLeft = typeof value === 'number' ? `${value * 4}px` : value;
        styles.paddingRight = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'py') {
        styles.paddingTop = typeof value === 'number' ? `${value * 4}px` : value;
        styles.paddingBottom = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'borderRadius') {
        styles.borderRadius =
          value === 'md' ? '6px' : value === 'lg' ? '8px' : value === 'full' ? '50%' : value;
      } else if (key === 'shadow') {
        styles.boxShadow =
          value === 'md'
            ? '0 4px 6px -1px rgba(0, 0, 0, 0.1)'
            : value === 'sm'
              ? '0 1px 2px 0 rgba(0, 0, 0, 0.05)'
              : value;
      } else if (key === 'bg') {
        styles.backgroundColor = value;
      } else if (key === 'color') {
        styles.color = value;
      } else if (key === 'border') {
        styles.border = value;
      } else if (key === 'borderColor') {
        styles.borderColor = value;
      } else if (key === 'borderTop') {
        styles.borderTop = value;
      } else if (key === 'borderBottom') {
        styles.borderBottom = value;
      } else if (key === 'borderLeft') {
        styles.borderLeft = value;
      } else if (key === 'borderRight') {
        styles.borderRight = value;
      } else if (key === 'position') {
        styles.position = value;
      } else if (key === 'top') {
        styles.top = value;
      } else if (key === 'zIndex') {
        styles.zIndex = value;
      } else if (key === 'display') {
        styles.display = value;
      } else if (key === 'alignItems') {
        styles.alignItems = value;
      } else if (key === 'justifyContent') {
        styles.justifyContent = value;
      } else if (key === 'textAlign') {
        styles.textAlign = value;
      } else if (key === 'cursor') {
        styles.cursor = value;
      } else if (key === 'minW') {
        styles.minWidth = value;
      } else if (key === 'minH') {
        styles.minHeight = value;
      } else if (key === 'spacing') {
        styles.gap = typeof value === 'number' ? `${value * 4}px` : value;
      } else if (key === 'wrap') {
        styles.flexWrap = value;
      } else if (key === 'isCentered') {
        if (value) {
          styles.display = 'flex';
          styles.alignItems = 'center';
          styles.justifyContent = 'center';
        }
      } else if (key === 'closeOnOverlayClick') {
        // This would be handled by the modal component
      } else if (key === 'scrollBehavior') {
        styles.overflow = value === 'inside' ? 'auto' : 'visible';
      } else if (key === 'size') {
        if (value === 'xl') {
          styles.maxWidth = '1200px';
          styles.width = '90vw';
        } else if (value === 'lg') {
          styles.maxWidth = '800px';
          styles.width = '80vw';
        } else if (value === 'md') {
          styles.maxWidth = '600px';
          styles.width = '70vw';
        }
      } else if (key.startsWith('_')) {
        // Handle pseudo-selectors like _hover
        // This would need special handling in a real implementation
      } else {
        // Convert kebab-case to camelCase for CSS properties
        const camelKey = key.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
        (styles as any)[camelKey] = value;
      }
    });

    return styles;
  };

  // Render individual component based on type
  const renderComponent = (componentName: string, component: any) => {
    const { type, description, props = {}, chakra_props = {} } = component;
    const componentProps = { ...props, ...chakra_props };
    const styles = convertPropsToStyles(componentProps);

    console.log(`[DynamicUIRenderer] Rendering ${componentName}:`, {
      type,
      description,
      styles: Object.keys(styles),
      props: Object.keys(componentProps)
    });

    // Get content for this component
    const componentContent =
      content[componentName] || content[componentName.replace(/_/g, '')] || {};

    switch (type) {
      case 'Box':
        return (
          <div key={componentName} style={styles} title={description}>
            {componentName === 'header' && (
              <>
                <div
                  style={{
                    fontSize: '20px',
                    fontWeight: 'bold',
                    marginBottom: '12px',
                    textAlign: 'left',
                    textTransform: 'uppercase',
                    letterSpacing: '1px',
                    color: theme.primary_color || '#fff'
                  }}
                >
                  {campaignTitle}
                </div>
                <div
                  style={{
                    fontSize: '13px',
                    opacity: 0.9,
                    marginBottom: '20px',
                    textAlign: 'left',
                    lineHeight: '1.5',
                    color: theme.text_color || '#ccc'
                  }}
                >
                  {campaignDescription}
                </div>
                {content.banner && (
                  <div
                    style={{
                      marginBottom: '20px',
                      fontSize: '15px',
                      textAlign: 'left',
                      fontWeight: '500',
                      color: theme.accent_color || '#fff'
                    }}
                  >
                    {content.banner}
                  </div>
                )}
                {content.cta && (
                  <button
                    style={{
                      background: theme.primary_color || '#fff',
                      color: theme.text_color || '#000',
                      border: '2px solid #333',
                      padding: '14px 28px',
                      fontSize: '14px',
                      cursor: 'pointer',
                      borderRadius: '0px',
                      fontWeight: 'bold',
                      transition: 'all 0.3s',
                      minHeight: '48px',
                      minWidth: '140px',
                      textAlign: 'left',
                      textTransform: 'uppercase',
                      letterSpacing: '1px',
                      boxShadow: '0 2px 4px rgba(0,0,0,0.2)'
                    }}
                    onMouseEnter={e => {
                      e.currentTarget.style.background = theme.accent_color || '#f0f0f0';
                      e.currentTarget.style.transform = 'translateY(-3px)';
                      e.currentTarget.style.boxShadow = '0 4px 8px rgba(0,0,0,0.3)';
                    }}
                    onMouseLeave={e => {
                      e.currentTarget.style.background = theme.primary_color || '#fff';
                      e.currentTarget.style.transform = 'translateY(0)';
                      e.currentTarget.style.boxShadow = '0 2px 4px rgba(0,0,0,0.2)';
                    }}
                  >
                    {content.cta}
                  </button>
                )}
              </>
            )}
            {componentName === 'product_card' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>
                  {componentContent.title || 'Product Name'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.description || 'Product description goes here'}
                </div>
                <div style={{ color: theme.primary_color || '#0f0', fontWeight: 'bold' }}>
                  ${componentContent.price || '99.99'}
                </div>
              </div>
            )}
            {componentName === 'shopping_cart' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '12px' }}>Shopping Cart</div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.item_count || '0'} items
                </div>
                <div style={{ fontWeight: 'bold' }}>Total: ${componentContent.total || '0.00'}</div>
              </div>
            )}
            {componentName === 'story_creator' && (
              <div style={{ textAlign: 'center' }}>
                <div style={{ fontSize: '14px', marginBottom: '8px' }}>+</div>
                <div style={{ fontSize: '11px' }}>Create Story</div>
              </div>
            )}
            {componentName === 'story_ring' && (
              <div style={{ textAlign: 'center' }}>
                <div
                  style={{
                    width: '50px',
                    height: '50px',
                    borderRadius: '50%',
                    background:
                      'linear-gradient(45deg, #f09433 0%,#e6683c 25%,#dc2743 50%,#cc2366 75%,#bc1888 100%)',
                    margin: '0 auto'
                  }}
                ></div>
              </div>
            )}
            {componentName === 'like_button' && (
              <button style={{ ...styles, background: 'transparent', border: 'none' }}>
                ❤️ {componentContent.likes || '0'}
              </button>
            )}
            {componentName === 'comment_section' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Comments</div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.comment_count || '0'} comments
                </div>
                <input
                  type="text"
                  placeholder="Add a comment..."
                  style={{
                    width: '100%',
                    padding: '8px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '12px'
                  }}
                />
              </div>
            )}
            {componentName === 'author_card' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                  {componentContent.author_name || 'Author Name'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.bio || 'Author bio goes here'}
                </div>
                <button
                  style={{
                    background: theme.primary_color || '#0f0',
                    color: '#fff',
                    border: 'none',
                    padding: '4px 8px',
                    fontSize: '10px',
                    borderRadius: '4px',
                    cursor: 'pointer'
                  }}
                >
                  Follow
                </button>
              </div>
            )}
            {componentName === 'author_bio' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>
                  {componentContent.author_name || 'Author Name'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.bio || 'Author bio goes here'}
                </div>
                <div style={{ fontSize: '10px', color: '#666' }}>
                  {componentContent.followers || '0'} followers
                </div>
              </div>
            )}
            {componentName === 'post_card' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>
                  {componentContent.title || 'Post Title'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.excerpt || 'Post excerpt goes here...'}
                </div>
                <div style={{ fontSize: '10px', color: '#666' }}>
                  {componentContent.reading_time || '5 min read'}
                </div>
              </div>
            )}
            {componentName === 'post_editor' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Editor</div>
                <textarea
                  placeholder={componentContent.placeholder || 'Start writing...'}
                  style={{
                    width: '100%',
                    minHeight: '200px',
                    padding: '12px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '12px',
                    fontFamily: 'inherit'
                  }}
                />
              </div>
            )}
            {componentName === 'search_bar' && (
              <div>
                <input
                  type="text"
                  placeholder={componentContent.placeholder || 'Search...'}
                  style={{
                    width: '100%',
                    padding: '8px 12px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '12px'
                  }}
                />
              </div>
            )}
            {componentName === 'pagination' && (
              <div style={{ display: 'flex', gap: '8px', justifyContent: 'center' }}>
                <button
                  style={{ padding: '4px 8px', border: '1px solid #ddd', background: '#fff' }}
                >
                  ←
                </button>
                <button
                  style={{ padding: '4px 8px', border: '1px solid #ddd', background: '#fff' }}
                >
                  1
                </button>
                <button
                  style={{ padding: '4px 8px', border: '1px solid #ddd', background: '#fff' }}
                >
                  2
                </button>
                <button
                  style={{ padding: '4px 8px', border: '1px solid #ddd', background: '#fff' }}
                >
                  3
                </button>
                <button
                  style={{ padding: '4px 8px', border: '1px solid #ddd', background: '#fff' }}
                >
                  →
                </button>
              </div>
            )}
            {componentName === 'related_posts' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Related Posts</div>
                <div style={{ fontSize: '11px' }}>
                  {componentContent.posts?.map((post: any, index: number) => (
                    <div
                      key={index}
                      style={{ marginBottom: '4px', padding: '4px', border: '1px solid #eee' }}
                    >
                      {post.title || `Related Post ${index + 1}`}
                    </div>
                  )) || 'No related posts'}
                </div>
              </div>
            )}
            {componentName === 'category_list' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Categories</div>
                <div style={{ fontSize: '11px' }}>
                  {componentContent.categories?.map((category: any, index: number) => (
                    <div
                      key={index}
                      style={{ marginBottom: '2px', padding: '2px 4px', background: '#f5f5f5' }}
                    >
                      {category.name || `Category ${index + 1}`} ({category.count || 0})
                    </div>
                  )) || 'No categories'}
                </div>
              </div>
            )}
            {componentName === 'tag_cloud' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Tags</div>
                <div style={{ fontSize: '11px', display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                  {componentContent.tags?.map((tag: any, index: number) => (
                    <span
                      key={index}
                      style={{
                        padding: '2px 6px',
                        background: '#e0e0e0',
                        borderRadius: '12px',
                        fontSize: '10px'
                      }}
                    >
                      {tag.name || `tag${index + 1}`}
                    </span>
                  )) || 'No tags'}
                </div>
              </div>
            )}
            {componentName === 'clap_button' && (
              <button style={{ ...styles, background: 'transparent', border: '1px solid #ddd' }}>
                👏 {componentContent.claps || '0'}
              </button>
            )}
            {componentName === 'reading_list' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Reading List</div>
                <div style={{ fontSize: '11px' }}>
                  {componentContent.items?.map((item: any, index: number) => (
                    <div
                      key={index}
                      style={{ marginBottom: '4px', padding: '4px', border: '1px solid #eee' }}
                    >
                      {item.title || `Article ${index + 1}`}
                    </div>
                  )) || 'No items'}
                </div>
              </div>
            )}
            {componentName === 'post_modal' && (
              <div style={{ ...styles, position: 'relative' }}>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Post Modal</div>
                <div style={{ fontSize: '11px' }}>{componentContent.title || 'Modal Content'}</div>
                <button
                  style={{
                    position: 'absolute',
                    top: '8px',
                    right: '8px',
                    background: '#f0f0f0',
                    border: 'none',
                    padding: '4px 8px',
                    cursor: 'pointer'
                  }}
                >
                  ×
                </button>
              </div>
            )}
            {componentName === 'explore_grid' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Explore</div>
                <div style={{ fontSize: '11px' }}>
                  {componentContent.items?.map((item: any, index: number) => (
                    <div
                      key={index}
                      style={{
                        marginBottom: '4px',
                        padding: '4px',
                        border: '1px solid #eee',
                        background: '#f9f9f9'
                      }}
                    >
                      {item.title || `Explore Item ${index + 1}`}
                    </div>
                  )) || 'No items to explore'}
                </div>
              </div>
            )}
            {componentName === 'feed_grid' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Feed</div>
                <div style={{ fontSize: '11px' }}>
                  {componentContent.posts?.map((post: any, index: number) => (
                    <div
                      key={index}
                      style={{
                        marginBottom: '4px',
                        padding: '4px',
                        border: '1px solid #eee',
                        background: '#f9f9f9'
                      }}
                    >
                      {post.title || `Post ${index + 1}`}
                    </div>
                  )) || 'No posts'}
                </div>
              </div>
            )}
            {componentName === 'article_card' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>
                  {componentContent.title || 'Article Title'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.excerpt || 'Article excerpt goes here...'}
                </div>
                <div style={{ fontSize: '10px', color: '#666' }}>
                  {componentContent.author || 'Author'} •{' '}
                  {componentContent.reading_time || '5 min read'}
                </div>
              </div>
            )}
            {componentName === 'article_editor' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Article Editor</div>
                <textarea
                  placeholder={componentContent.placeholder || 'Tell your story...'}
                  style={{
                    width: '100%',
                    minHeight: '300px',
                    padding: '12px',
                    border: '1px solid #ddd',
                    borderRadius: '4px',
                    fontSize: '12px',
                    fontFamily: 'inherit'
                  }}
                />
              </div>
            )}
            {componentName === 'hero_section' && (
              <div style={{ textAlign: 'center', padding: '40px 20px' }}>
                <div style={{ fontSize: '24px', fontWeight: 'bold', marginBottom: '12px' }}>
                  {componentContent.title || campaignTitle}
                </div>
                <div style={{ fontSize: '16px', marginBottom: '20px', opacity: 0.8 }}>
                  {componentContent.subtitle || campaignDescription}
                </div>
                {componentContent.cta_text && (
                  <button
                    style={{
                      background: theme.primary_color || '#0f0',
                      color: '#fff',
                      border: 'none',
                      padding: '12px 24px',
                      fontSize: '14px',
                      borderRadius: '6px',
                      cursor: 'pointer'
                    }}
                  >
                    {componentContent.cta_text}
                  </button>
                )}
              </div>
            )}
            {componentName === 'editor_interface' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Editor Interface</div>
                <div style={{ marginBottom: '8px' }}>
                  <textarea
                    placeholder={componentContent.placeholder || 'Start writing...'}
                    style={{
                      width: '100%',
                      minHeight: '200px',
                      padding: '12px',
                      border: '1px solid #ddd',
                      borderRadius: '4px',
                      fontSize: '12px',
                      fontFamily: 'inherit'
                    }}
                  />
                </div>
                {componentContent.toolbar_items && (
                  <div style={{ display: 'flex', gap: '4px', marginBottom: '8px' }}>
                    {componentContent.toolbar_items.map((item: string, index: number) => (
                      <button
                        key={index}
                        style={{
                          padding: '4px 8px',
                          border: '1px solid #ddd',
                          background: '#fff',
                          fontSize: '10px'
                        }}
                      >
                        {item}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}
            {componentName === 'article_metadata' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Article Metadata</div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Reading Time: {componentContent.reading_time || '5 min read'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Published: {componentContent.publish_date || 'Today'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Tags: {componentContent.tags?.join(', ') || 'No tags'}
                </div>
                {componentContent.engagement && (
                  <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                    Engagement: {componentContent.engagement.claps || 0} claps,{' '}
                    {componentContent.engagement.responses || 0} responses
                  </div>
                )}
              </div>
            )}
            {componentName === 'post_metadata' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Post Metadata</div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Author: {componentContent.author || 'Unknown'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Published: {componentContent.publish_date || 'Today'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Reading Time: {componentContent.reading_time || '5 min read'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Category: {componentContent.category || 'Uncategorized'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                  Tags: {componentContent.tags?.join(', ') || 'No tags'}
                </div>
                {componentContent.engagement && (
                  <div style={{ fontSize: '11px', marginBottom: '4px' }}>
                    Views: {componentContent.engagement.views || 0} | Likes:{' '}
                    {componentContent.engagement.likes || 0} | Comments:{' '}
                    {componentContent.engagement.comments || 0}
                  </div>
                )}
              </div>
            )}
            {componentName === 'navigation_items' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Navigation</div>
                <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap' }}>
                  {componentContent.map((item: any, index: number) => (
                    <a
                      key={index}
                      href={item.path}
                      style={{
                        color: theme.primary_color || '#0f0',
                        textDecoration: 'none',
                        fontSize: '11px'
                      }}
                    >
                      {item.label}
                    </a>
                  ))}
                </div>
              </div>
            )}
            {componentName === 'sidebar_widgets' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Sidebar Widgets</div>
                {componentContent.map((widget: any, index: number) => (
                  <div
                    key={index}
                    style={{
                      marginBottom: '12px',
                      padding: '8px',
                      border: '1px solid #ddd',
                      background: '#f9f9f9'
                    }}
                  >
                    <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>{widget.title}</div>
                    <div style={{ fontSize: '11px' }}>
                      {widget.type === 'recent_posts' && `Show ${widget.count || 5} recent posts`}
                      {widget.type === 'categories' && 'Category list with counts'}
                      {widget.type === 'tags' && `Show ${widget.count || 20} popular tags`}
                      {widget.type === 'newsletter' && 'Newsletter signup form'}
                    </div>
                  </div>
                ))}
              </div>
            )}
            {componentName === 'story_interface' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Story Interface</div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  Camera Options: {componentContent.camera_options?.join(', ') || 'photo, video'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  Filters: {componentContent.filters?.length || 0} available
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  Text Tools: {componentContent.text_tools?.join(', ') || 'font, color'}
                </div>
                <div style={{ fontSize: '11px' }}>
                  Stickers: {componentContent.sticker_options?.length || 0} options
                </div>
              </div>
            )}
            {componentName === 'post_creation' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Post Creation</div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  Aspect Ratios: {componentContent.aspect_ratios?.join(', ') || '1:1, 4:5'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  Editing Tools: {componentContent.editing_tools?.join(', ') || 'crop, rotate'}
                </div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  Caption: {componentContent.caption_placeholder || 'Write a caption...'}
                </div>
                <div style={{ fontSize: '11px' }}>
                  Features: {componentContent.hashtag_suggestions ? 'Hashtag suggestions' : ''}
                  {componentContent.location_tagging ? ', Location tagging' : ''}
                  {componentContent.alt_text ? ', Alt text' : ''}
                </div>
              </div>
            )}
            {componentName === 'engagement_metrics' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Engagement Metrics</div>
                <div
                  style={{
                    fontSize: '11px',
                    display: 'grid',
                    gridTemplateColumns: 'repeat(2, 1fr)',
                    gap: '8px'
                  }}
                >
                  <div>Likes: {componentContent.likes || 0}</div>
                  <div>Comments: {componentContent.comments || 0}</div>
                  <div>Shares: {componentContent.shares || 0}</div>
                  <div>Saves: {componentContent.saves || 0}</div>
                  <div>Views: {componentContent.views || 0}</div>
                </div>
              </div>
            )}
            {componentName === 'lead_form' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Lead Form</div>
                {componentContent.fields?.map((field: string, index: number) => (
                  <div key={index} style={{ marginBottom: '8px' }}>
                    <input
                      type={field === 'email' ? 'email' : 'text'}
                      placeholder={field.charAt(0).toUpperCase() + field.slice(1).replace('_', ' ')}
                      style={{
                        width: '100%',
                        padding: '8px',
                        border: '1px solid #ddd',
                        borderRadius: '4px',
                        fontSize: '12px'
                      }}
                    />
                  </div>
                ))}
                <button
                  style={{
                    background: theme.primary_color || '#0f0',
                    color: '#fff',
                    border: 'none',
                    padding: '8px 16px',
                    borderRadius: '4px',
                    cursor: 'pointer',
                    fontSize: '12px'
                  }}
                >
                  {componentContent.submit_text || 'Submit'}
                </button>
              </div>
            )}
            {componentName === 'architecture_overview' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Architecture Overview</div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.description || 'Architecture description goes here'}
                </div>
                <div style={{ fontSize: '11px' }}>
                  Sections: {componentContent.sections?.join(', ') || 'No sections'}
                </div>
              </div>
            )}
            {componentName === 'site_onboarding' && (
              <div>
                <div style={{ fontWeight: 'bold', marginBottom: '8px' }}>Site Onboarding</div>
                <div style={{ fontSize: '11px', marginBottom: '8px' }}>
                  {componentContent.main_text || "Welcome! Let's get started."}
                </div>
                <div style={{ fontSize: '11px' }}>
                  Path: {componentContent.path || '/onboarding'}
                </div>
                {componentContent.steps && (
                  <div style={{ marginTop: '8px' }}>
                    {componentContent.steps.map((step: any, index: number) => (
                      <div
                        key={index}
                        style={{
                          marginBottom: '4px',
                          padding: '4px',
                          border: '1px solid #ddd',
                          background: '#f9f9f9'
                        }}
                      >
                        Step {step.step}: {step.text}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
            {/* Default case for unknown components */}
            {!['Box', 'Grid', 'VStack', 'HStack', 'Flex', 'InputGroup', 'Modal', 'Button'].includes(
              type
            ) && (
              <div style={{ fontSize: '11px', color: '#666' }}>
                {componentName}: {description || 'Component description not available'}
              </div>
            )}
          </div>
        );

      case 'Grid':
        return (
          <div key={componentName} style={{ ...styles, display: 'grid' }} title={description}>
            {componentName === 'product_grid' && (
              <>
                {Array.from({ length: 4 }, (_, i) => (
                  <div
                    key={i}
                    style={{
                      padding: '8px',
                      border: '1px solid #ddd',
                      background: '#fff',
                      borderRadius: '4px'
                    }}
                  >
                    <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>Product {i + 1}</div>
                    <div style={{ fontSize: '11px', color: '#666' }}>
                      ${(99.99 + i * 10).toFixed(2)}
                    </div>
                  </div>
                ))}
              </>
            )}
            {componentName === 'feed_grid' && (
              <>
                {Array.from({ length: 6 }, (_, i) => (
                  <div
                    key={i}
                    style={{
                      padding: '8px',
                      border: '1px solid #ddd',
                      background: '#fff',
                      borderRadius: '4px'
                    }}
                  >
                    <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>Post {i + 1}</div>
                    <div style={{ fontSize: '11px', color: '#666' }}>{i + 1} likes</div>
                  </div>
                ))}
              </>
            )}
            {componentName === 'explore_grid' && (
              <>
                {Array.from({ length: 6 }, (_, i) => (
                  <div
                    key={i}
                    style={{
                      padding: '8px',
                      border: '1px solid #ddd',
                      background: '#fff',
                      borderRadius: '4px'
                    }}
                  >
                    <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>Explore {i + 1}</div>
                    <div style={{ fontSize: '11px', color: '#666' }}>Trending</div>
                  </div>
                ))}
              </>
            )}
            {componentName === 'related_posts' && (
              <>
                {Array.from({ length: 3 }, (_, i) => (
                  <div
                    key={i}
                    style={{
                      padding: '8px',
                      border: '1px solid #ddd',
                      background: '#fff',
                      borderRadius: '4px'
                    }}
                  >
                    <div style={{ fontWeight: 'bold', marginBottom: '4px' }}>
                      Related Post {i + 1}
                    </div>
                    <div style={{ fontSize: '11px', color: '#666' }}>5 min read</div>
                  </div>
                ))}
              </>
            )}
            {/* Default grid content */}
            {!['product_grid', 'feed_grid', 'explore_grid', 'related_posts'].includes(
              componentName
            ) && (
              <div style={{ fontSize: '11px', color: '#666' }}>
                Grid: {description || 'Grid component'}
              </div>
            )}
          </div>
        );

      case 'VStack':
        return (
          <div
            key={componentName}
            style={{ ...styles, display: 'flex', flexDirection: 'column' }}
            title={description}
          >
            <div style={{ fontSize: '11px', color: '#666' }}>
              VStack: {description || 'Vertical stack component'}
            </div>
          </div>
        );

      case 'HStack':
        return (
          <div
            key={componentName}
            style={{ ...styles, display: 'flex', flexDirection: 'row' }}
            title={description}
          >
            <div style={{ fontSize: '11px', color: '#666' }}>
              HStack: {description || 'Horizontal stack component'}
            </div>
          </div>
        );

      case 'Flex':
        return (
          <div key={componentName} style={{ ...styles, display: 'flex' }} title={description}>
            <div style={{ fontSize: '11px', color: '#666' }}>
              Flex: {description || 'Flex component'}
            </div>
          </div>
        );

      case 'InputGroup':
        return (
          <div key={componentName} style={styles} title={description}>
            <input
              type="text"
              placeholder={componentContent.placeholder || 'Search...'}
              style={{
                width: '100%',
                padding: '8px 12px',
                border: '1px solid #ddd',
                borderRadius: '4px',
                fontSize: '12px'
              }}
            />
          </div>
        );

      case 'Modal':
        return (
          <div key={componentName} style={{ ...styles, position: 'relative' }} title={description}>
            <div style={{ fontSize: '11px', color: '#666' }}>
              Modal: {description || 'Modal component'}
            </div>
          </div>
        );

      case 'Button':
        return (
          <button key={componentName} style={styles} title={description}>
            {componentContent.text || componentName.replace(/_/g, ' ').toUpperCase()}
          </button>
        );

      default:
        return (
          <div key={componentName} style={styles} title={description}>
            <div style={{ fontSize: '11px', color: '#666' }}>
              {type}: {description || 'Unknown component type'}
            </div>
          </div>
        );
    }
  };

  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))',
        gap: '16px',
        padding: '20px',
        background: theme.background_color || '#000',
        minHeight: '400px'
      }}
    >
      {Object.entries(components).map(([componentName, component]) => (
        <div
          key={componentName}
          style={{
            background: theme.background_color || '#111',
            border: `1px solid ${theme.border_color || '#333'}`,
            borderRadius: theme.border_radius || '8px',
            padding: '16px',
            transition: 'all 0.2s ease',
            cursor: 'pointer'
          }}
          onMouseOver={e => {
            e.currentTarget.style.borderColor = theme.primary_color || '#0f0';
            e.currentTarget.style.boxShadow = `0 0 8px ${theme.primary_color || '#0f0'}33`;
          }}
          onMouseOut={e => {
            e.currentTarget.style.borderColor = theme.border_color || '#333';
            e.currentTarget.style.boxShadow = 'none';
          }}
        >
          <div
            style={{
              fontSize: '12px',
              fontWeight: 'bold',
              color: theme.primary_color || '#0f0',
              marginBottom: '8px',
              textTransform: 'uppercase',
              letterSpacing: '0.5px'
            }}
          >
            {componentName.replace(/_/g, ' ')}
          </div>
          {renderComponent(componentName, component)}
        </div>
      ))}
    </div>
  );
};

interface CampaignUIRendererProps {
  campaign?: CampaignMetadata;
  isLoading?: boolean;
}

const CampaignUIRenderer: React.FC<CampaignUIRendererProps> = ({ campaign, isLoading = false }) => {
  console.log('[CampaignUIRenderer] Render called:', {
    campaign: campaign
      ? {
          id: campaign.campaignId || (campaign as any).id || (campaign as any).slug,
          title: campaign.title || (campaign as any).title,
          status: campaign.status || (campaign as any).status,
          hasServiceSpecific: !!campaign.serviceSpecific,
          hasCampaign: !!campaign.serviceSpecific?.campaign,
          features: campaign.features?.length || 0,
          hasMetadata: !!(campaign as any).metadata,
          hasServiceSpecificCampaign: !!(campaign as any).metadata?.service_specific?.campaign,
          // Debug: Check for ui_components in different locations
          hasUIComponentsInCampaign: !!(campaign as any).ui_components,
          hasUIComponentsInServiceSpecific: !!(campaign as any).serviceSpecific?.campaign
            ?.ui_components,
          hasUIComponentsInMetadata: !!(campaign as any).metadata?.service_specific?.campaign
            ?.ui_components,
          campaignKeys: Object.keys(campaign),
          serviceSpecificKeys: campaign.serviceSpecific
            ? Object.keys(campaign.serviceSpecific)
            : [],
          metadataKeys: (campaign as any).metadata ? Object.keys((campaign as any).metadata) : []
        }
      : null,
    isLoading
  });

  if (isLoading) {
    console.log('[CampaignUIRenderer] Rendering loading state');
    return (
      <div
        style={{
          fontFamily: 'Monaco, Menlo, Consolas, monospace',
          background: '#000',
          color: '#fff',
          textAlign: 'center',
          padding: '40px',
          fontSize: '12px'
        }}
      >
        Loading campaign interface...
      </div>
    );
  }

  if (!campaign) {
    console.log('[CampaignUIRenderer] No campaign provided');
    return (
      <div
        style={{
          fontFamily: 'Monaco, Menlo, Consolas, monospace',
          background: '#000',
          color: '#666',
          textAlign: 'center',
          padding: '40px',
          fontSize: '12px'
        }}
      >
        <div>No campaign selected</div>
        <div>Select a campaign to view its interface</div>
      </div>
    );
  }

  // Extract data from campaign metadata with detailed logging
  // Handle both CampaignMetadata type and campaign.json structure
  const campaignData =
    (campaign as any).metadata?.service_specific?.campaign ||
    campaign.serviceSpecific?.campaign ||
    campaign; // Fallback to campaign itself if it's already the right structure

  // Extract UI components from multiple possible locations
  // Priority: nested structure (campaign.json) -> flat structure (default_campaign.json) -> fallback
  const uiContent =
    campaignData.ui_content || (campaign as any).ui_content || campaign.ui_content || {};

  const uiComponents =
    campaignData.ui_components ||
    (campaign as any).ui_components ||
    (campaign as any).ui_components ||
    {};

  const platformType =
    campaignData.platform_type ||
    campaignData.focus ||
    (campaign as any).platform_type ||
    (campaign as any).platform_type ||
    'general';

  const features =
    campaign.features || campaignData.features || (campaign as any).metadata?.features || [];

  const aboutContent = campaignData.about || campaign.about || (campaign as any).about || {};

  const theme = campaignData.theme || (campaign as any).theme || (campaign as any).theme || {};

  const serviceConfigs =
    campaignData.service_configs ||
    (campaign as any).service_configs ||
    (campaign as any).service_configs ||
    {};

  const campaignTitle =
    campaign.title || campaignData.title || (campaign as any).title || 'Campaign Interface';

  const campaignDescription =
    campaign.description ||
    campaignData.description ||
    (campaign as any).description ||
    'Welcome to your campaign dashboard';

  console.log('[CampaignUIRenderer] Extracted data:', {
    uiContent: Object.keys(uiContent),
    uiComponents: Object.keys(uiComponents),
    platformType,
    features: features.length,
    aboutContent: aboutContent.order?.length || 0,
    theme: Object.keys(theme),
    serviceConfigs: Object.keys(serviceConfigs),
    campaignDataKeys: Object.keys(campaignData),
    hasUIComponents: !!uiComponents && Object.keys(uiComponents).length > 0,
    uiComponentsType: typeof uiComponents,
    uiComponentsKeys: uiComponents ? Object.keys(uiComponents) : [],
    fullCampaignData: campaignData,
    // Debug: Check if ui_components exists in different locations
    debugUIComponents: {
      inCampaignData: !!campaignData.ui_components,
      inCampaign: !!(campaign as any).ui_components,
      inMetadata: !!(campaign as any).metadata?.service_specific?.campaign?.ui_components,
      campaignDataUIComponents: campaignData.ui_components,
      campaignUIComponents: (campaign as any).ui_components,
      metadataUIComponents: (campaign as any).metadata?.service_specific?.campaign?.ui_components
    }
  });

  // Apply theme styles to the main container
  const themeStyles = {
    fontFamily: theme.font_family || 'Monaco, Menlo, Consolas, monospace',
    background: theme.background_color || '#000',
    color: theme.text_color || '#fff',
    fontSize: '12px',
    lineHeight: '1.4'
  };

  return (
    <div style={themeStyles}>
      {/* Dynamic UI Components from campaign data - PRIORITY RENDERING */}
      {uiComponents && typeof uiComponents === 'object' && Object.keys(uiComponents).length > 0 ? (
        <div
          style={{
            minHeight: '400px',
            background: theme.background_color || '#000',
            color: theme.text_color || '#fff'
          }}
        >
          <DynamicUIRenderer
            components={uiComponents}
            content={uiContent}
            theme={theme}
            campaignTitle={campaignTitle}
            campaignDescription={campaignDescription}
          />
        </div>
      ) : (
        /* Fallback: Enhanced static interface when no UI components defined */
        <div
          style={{
            textAlign: 'center',
            padding: '40px 20px',
            borderBottom: '1px solid #333',
            background: theme.background_color || '#111',
            minHeight: '300px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'center',
            alignItems: 'center'
          }}
        >
          <div
            style={{
              fontSize: '24px',
              fontWeight: 'bold',
              marginBottom: '12px',
              color: theme.primary_color || '#fff'
            }}
          >
            {campaignTitle}
          </div>
          <div
            style={{
              fontSize: '14px',
              color: theme.text_color || '#ccc',
              marginBottom: '24px',
              maxWidth: '600px',
              lineHeight: '1.5'
            }}
          >
            {campaignDescription}
          </div>
          {uiContent.banner && (
            <div
              style={{
                marginBottom: '20px',
                color: theme.accent_color || '#ccc',
                fontSize: '16px',
                fontWeight: '500'
              }}
            >
              {uiContent.banner}
            </div>
          )}
          {uiContent.cta && (
            <button
              style={{
                background: theme.primary_color || '#fff',
                color: theme.background_color || '#000',
                border: `2px solid ${theme.primary_color || '#fff'}`,
                padding: '12px 24px',
                fontSize: '14px',
                fontWeight: 'bold',
                cursor: 'pointer',
                fontFamily: 'inherit',
                borderRadius: theme.border_radius || '6px',
                transition: 'all 0.2s ease',
                textTransform: 'uppercase',
                letterSpacing: '0.5px'
              }}
              onMouseOver={e => {
                e.currentTarget.style.background = 'transparent';
                e.currentTarget.style.color = theme.primary_color || '#fff';
              }}
              onMouseOut={e => {
                e.currentTarget.style.background = theme.primary_color || '#fff';
                e.currentTarget.style.color = theme.background_color || '#000';
              }}
            >
              {uiContent.cta}
            </button>
          )}
        </div>
      )}

      {/* Features */}
      {features.length > 0 && (
        <div
          style={{
            margin: '16px',
            padding: '16px',
            border: '1px solid #333',
            background: '#111'
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
            Available Features
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
            {features.map((feature, index) => (
              <span
                key={index}
                style={{
                  display: 'inline-block',
                  background: '#333',
                  color: '#fff',
                  padding: '2px 6px',
                  fontSize: '10px',
                  margin: '2px',
                  border: '1px solid #555'
                }}
              >
                {feature.replace('_', ' ').toUpperCase()}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Platform-specific content */}
      {platformType === 'social_media' && (
        <div style={{ padding: '16px' }}>
          <div
            style={{
              margin: '16px 0',
              padding: '16px',
              border: '1px solid #333',
              background: '#111'
            }}
          >
            <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
              Social Media Features
            </div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Post creation and sharing</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Real-time messaging</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Media upload and editing</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Community engagement tools</div>
          </div>
        </div>
      )}

      {platformType === 'blogging' && (
        <div style={{ padding: '16px' }}>
          <div
            style={{
              margin: '16px 0',
              padding: '16px',
              border: '1px solid #333',
              background: '#111'
            }}
          >
            <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
              Blogging Features
            </div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Rich text editor</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Draft management</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Publishing tools</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Content analytics</div>
          </div>
        </div>
      )}

      {platformType === 'marketplace' && (
        <div style={{ padding: '16px' }}>
          <div
            style={{
              margin: '16px 0',
              padding: '16px',
              border: '1px solid #333',
              background: '#111'
            }}
          >
            <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
              Marketplace Features
            </div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Product catalog</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Shopping cart</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Payment processing</div>
            <div style={{ color: '#ccc', marginBottom: '8px' }}>• Order management</div>
          </div>
        </div>
      )}

      {/* About Content */}
      {aboutContent.order && aboutContent.order.length > 0 && (
        <div
          style={{
            margin: '16px',
            padding: '16px',
            border: '1px solid #333',
            background: '#111'
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
            About This Campaign
          </div>
          {aboutContent.order.map((item: any, index: number) => (
            <div key={index} style={{ marginBottom: '16px' }}>
              {item.title && (
                <div style={{ color: '#fff', fontWeight: 'bold', marginBottom: '4px' }}>
                  {item.title}
                </div>
              )}
              {item.subtitle && (
                <div style={{ color: '#fff', fontWeight: 'bold', marginBottom: '4px' }}>
                  {item.subtitle}
                </div>
              )}
              {item.p && <div style={{ color: '#ccc', marginBottom: '8px' }}>{item.p}</div>}
              {item.list && item.list.length > 0 && (
                <div style={{ marginLeft: '16px' }}>
                  {item.list.map((listItem: string, listIndex: number) => (
                    <div key={listIndex} style={{ color: '#ccc', marginBottom: '4px' }}>
                      • {listItem}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Lead Form */}
      {uiContent.lead_form && (
        <div
          style={{
            margin: '16px',
            padding: '16px',
            border: '1px solid #333',
            background: '#111'
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
            Get Started
          </div>
          <div style={{ maxWidth: '400px', margin: '0 auto' }}>
            {uiContent.lead_form.fields?.map((field: string, index: number) => (
              <div key={index} style={{ marginBottom: '12px' }}>
                <label
                  style={{
                    display: 'block',
                    fontSize: '11px',
                    color: '#ccc',
                    marginBottom: '4px'
                  }}
                >
                  {field.charAt(0).toUpperCase() + field.slice(1).replace('_', ' ')}
                </label>
                <input
                  type="text"
                  placeholder={field.charAt(0).toUpperCase() + field.slice(1).replace('_', ' ')}
                  style={{
                    background: '#000',
                    color: '#fff',
                    border: '1px solid #333',
                    padding: '6px 8px',
                    fontSize: '11px',
                    fontFamily: 'inherit',
                    width: '100%'
                  }}
                />
              </div>
            ))}
            <button
              style={{
                background: '#fff',
                color: '#000',
                border: '1px solid #333',
                padding: '6px 12px',
                fontSize: '11px',
                cursor: 'pointer',
                fontFamily: 'inherit',
                width: '100%'
              }}
            >
              {uiContent.lead_form.submit_text || 'Submit'}
            </button>
          </div>
        </div>
      )}

      {/* Theme Information */}
      {Object.keys(theme).length > 0 && (
        <div
          style={{
            margin: '16px',
            padding: '16px',
            border: '1px solid #333',
            background: '#111'
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
            Campaign Theme
          </div>
          {theme.primary_color && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Primary Color:</strong>
              <span
                style={{
                  display: 'inline-block',
                  width: '20px',
                  height: '12px',
                  backgroundColor: theme.primary_color,
                  marginLeft: '8px',
                  border: '1px solid #333'
                }}
              />
              {theme.primary_color}
            </div>
          )}
          {theme.secondary_color && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Secondary Color:</strong>
              <span
                style={{
                  display: 'inline-block',
                  width: '20px',
                  height: '12px',
                  backgroundColor: theme.secondary_color,
                  marginLeft: '8px',
                  border: '1px solid #333'
                }}
              />
              {theme.secondary_color}
            </div>
          )}
          {theme.accent_color && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Accent Color:</strong>
              <span
                style={{
                  display: 'inline-block',
                  width: '20px',
                  height: '12px',
                  backgroundColor: theme.accent_color,
                  marginLeft: '8px',
                  border: '1px solid #333'
                }}
              />
              {theme.accent_color}
            </div>
          )}
          {theme.background_color && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Background:</strong> {theme.background_color}
            </div>
          )}
          {theme.text_color && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Text Color:</strong> {theme.text_color}
            </div>
          )}
          {theme.font_family && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Font Family:</strong> {theme.font_family}
            </div>
          )}
          {theme.border_radius && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Border Radius:</strong> {theme.border_radius}
            </div>
          )}
          {theme.shadow && (
            <div style={{ color: '#ccc', marginBottom: '8px' }}>
              <strong>Shadow:</strong> {theme.shadow}
            </div>
          )}
        </div>
      )}

      {/* UI Components */}
      {uiComponents && Object.keys(uiComponents).length > 0 && (
        <div
          style={{
            margin: '16px',
            padding: '16px',
            border: '1px solid #333',
            background: '#111'
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
            UI Components ({Object.keys(uiComponents).length})
          </div>
          {Object.entries(uiComponents).map(([componentName, component]: [string, any]) => (
            <div
              key={componentName}
              style={{
                marginBottom: '12px',
                padding: '8px',
                border: '1px solid #333',
                background: '#000'
              }}
            >
              <div style={{ color: '#fff', fontWeight: 'bold', marginBottom: '4px' }}>
                {componentName.replace(/_/g, ' ').toUpperCase()}
              </div>
              {component.type && (
                <div style={{ color: '#0f0', fontSize: '11px', marginBottom: '4px' }}>
                  Type: {component.type}
                </div>
              )}
              {component.description && (
                <div style={{ color: '#ccc', fontSize: '11px', marginBottom: '4px' }}>
                  {component.description}
                </div>
              )}
              {component.chakra_props && (
                <div style={{ color: '#888', fontSize: '10px' }}>
                  Chakra Props: {Object.keys(component.chakra_props).length} configured
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Service Configurations */}
      {Object.keys(serviceConfigs).length > 0 && (
        <div
          style={{
            margin: '16px',
            padding: '16px',
            border: '1px solid #333',
            background: '#111'
          }}
        >
          <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
            Service Configurations
          </div>
          {Object.entries(serviceConfigs).map(([serviceName, config]: [string, any]) => (
            <div key={serviceName} style={{ marginBottom: '8px' }}>
              <div style={{ color: '#fff', fontWeight: 'bold' }}>
                {serviceName.toUpperCase()}: {config.enabled ? 'ENABLED' : 'DISABLED'}
              </div>
              {config.description && (
                <div style={{ color: '#ccc', fontSize: '11px' }}>{config.description}</div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Campaign Status */}
      <div
        style={{
          margin: '16px',
          padding: '16px',
          border: '1px solid #333',
          background: '#111'
        }}
      >
        <div style={{ fontSize: '14px', fontWeight: 'bold', marginBottom: '12px' }}>
          Campaign Status
        </div>
        <div style={{ color: '#ccc', marginBottom: '8px' }}>
          <strong>Status:</strong> {campaign.status || 'Unknown'}
        </div>
        <div style={{ color: '#ccc', marginBottom: '8px' }}>
          <strong>Platform Type:</strong> {platformType}
        </div>
        <div style={{ color: '#ccc', marginBottom: '8px' }}>
          <strong>Features:</strong> {features.length} enabled
        </div>
        <div style={{ color: '#ccc', marginBottom: '8px' }}>
          <strong>UI Components:</strong> {Object.keys(uiContent).length} configured
        </div>
        <div style={{ color: '#ccc', marginBottom: '8px' }}>
          <strong>Service Configs:</strong> {Object.keys(serviceConfigs).length} services
        </div>
      </div>
    </div>
  );
};

export default CampaignUIRenderer;
