<script>
  import { onMount } from 'svelte';
  import { PreviewArticle, RenderFromPreview, ProcessArticleToVideo, OpenFolder } from '../wailsjs/go/main/App.js';
  import { EventsOn } from '../wailsjs/runtime/runtime.js';

  let articleUrl = 'https://vnexpress.net/yeu-cau-thu-hoc-phi-dai-hoc-theo-thang-5139356.html';
  let headline1 = '';
  let headline2 = '';
  let fullScript = '';
  let images = [];
  let taskId = '';
  let voiceName = 'vi-VN-HoaiMyNeural';
  let ratio = '9:16';
  let footerColor = '#EA580C'; // Default: Vibrant News Orange

  // Presets
  const colorPresets = [
    { label: 'Cam Nóng', hex: '#EA580C' },
    { label: 'Vàng Đậm', hex: '#D97706' },
    { label: 'Đỏ Thời Sự', hex: '#DC2626' },
    { label: 'Xanh Navy', hex: '#1D4ED8' },
    { label: 'Tím Đậm', hex: '#7C3AED' },
    { label: 'Đen Slate', hex: '#0F172A' },
  ];

  let isCrawling = false;
  let isRendering = false;
  let progressStep = 'idle';
  let progressPercent = 0;
  let progressLog = 'Sẵn sàng nhận link bài báo...';
  let completedVideo = '';
  let errorMessage = '';

  const voices = [
    { id: 'vi-VN-HoaiMyNeural', name: 'Hoài My', desc: 'Nữ thời sự truyền cảm', tag: 'Edge-TTS' },
    { id: 'vi-VN-NamMinhNeural', name: 'Nam Minh', desc: 'Nam thời sự dứt khoát', tag: 'Edge-TTS' },
    { id: 'hn_female_ngochuyen_full_48k-fhg', name: 'Ngọc Huyền', desc: 'Nữ Hà Nội chuẩn tin tức', tag: 'Vbee AI' }
  ];

  onMount(() => {
    EventsOn('pipeline:progress', (data) => {
      progressStep = data.step;
      progressPercent = data.percent;
      progressLog = data.log;
      if (data.isComplete && data.videoPath) {
        completedVideo = data.videoPath;
        isRendering = false;
      }
    });
  });

  async function handlePreview() {
    if (!articleUrl.trim()) {
      errorMessage = 'Vui lòng dán đường link bài báo hợp lệ.';
      return;
    }
    errorMessage = '';
    isCrawling = true;
    progressPercent = 10;
    progressLog = 'Đang phân tích trang báo và tải ảnh chất lượng cao...';

    try {
      const data = await PreviewArticle(articleUrl);
      taskId = data.taskId;
      headline1 = data.headline1 || '';
      headline2 = data.headline2 || '';
      if (!headline1 && !headline2) {
        headline1 = data.headline || '';
      }
      fullScript = data.fullScript || '';
      images = data.images || [];
      isCrawling = false;
      progressStep = 'preview_ready';
      progressPercent = 100;
      progressLog = 'Bóc tách thành công! Bạn có thể kiểm tra kịch bản & ảnh bên dưới.';
    } catch (err) {
      errorMessage = String(err);
      isCrawling = false;
      progressStep = 'error';
    }
  }

  async function handleRender() {
    if (!headline1.trim() && !headline2.trim()) {
      errorMessage = 'Tiêu đề bản tin không được để trống.';
      return;
    }
    if (!fullScript.trim()) {
      errorMessage = 'Kịch bản bản tin không được để trống.';
      return;
    }
    if (images.length === 0) {
      errorMessage = 'Cần ít nhất 1 hình ảnh minh họa để dựng video.';
      return;
    }

    errorMessage = '';
    completedVideo = '';
    isRendering = true;
    progressPercent = 15;
    progressLog = 'Bắt đầu tổng hợp giọng đọc & dựng video...';

    try {
      const res = await RenderFromPreview({
        taskId: taskId,
        headline1: headline1,
        headline2: headline2,
        headline: headline1 + (headline2 ? ' - ' + headline2 : ''),
        footerColor: footerColor,
        tagText: '● TIN NÓNG HÔM NAY',
        script: fullScript,
        images: images,
        voiceName: voiceName,
        ratio: ratio
      });
      if (res && res.videoPath) {
        completedVideo = res.videoPath;
      }
    } catch (err) {
      errorMessage = String(err);
      isRendering = false;
      progressStep = 'error';
    }
  }

  async function handleQuickRun() {
    if (!articleUrl.trim()) {
      errorMessage = 'Vui lòng nhập đường link bài báo.';
      return;
    }
    errorMessage = '';
    completedVideo = '';
    isRendering = true;
    progressPercent = 5;
    progressLog = 'Đang khởi chạy luồng tự động 1-Click...';

    try {
      const res = await ProcessArticleToVideo(articleUrl, voiceName, headline1, footerColor);
      if (res && res.videoPath) {
        completedVideo = res.videoPath;
      }
    } catch (err) {
      errorMessage = String(err);
      isRendering = false;
      progressStep = 'error';
    }
  }

  function toggleImage(img) {
    if (images.includes(img)) {
      if (images.length <= 1) {
        alert('Cần giữ lại ít nhất 1 ảnh để làm video!');
        return;
      }
      images = images.filter(i => i !== img);
    } else {
      images = [...images, img];
    }
  }

  function openOutputFolder() {
    OpenFolder(completedVideo);
  }
</script>

<div class="studio-app">
  <!-- Top Navigation Bar -->
  <header class="top-nav">
    <div class="nav-left">
      <div class="logo-mark">
        <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <polygon points="23 7 16 12 23 17 23 7"></polygon>
          <rect x="1" y="5" width="15" height="14" rx="2" ry="2"></rect>
        </svg>
      </div>
      <div class="brand-text">
        <span class="app-title">GoNews Video Studio</span>
        <span class="app-badge">Format Theanh28 / Beatvn Pro</span>
      </div>
    </div>

    <div class="nav-center">
      <div class="status-pill">
        <span class="pulse-indicator {isCrawling || isRendering ? 'active' : ''}"></span>
        <span class="status-text">
          {#if isCrawling}
            Đang phân tích bài báo...
          {:else if isRendering}
            Đang dựng FFmpeg ({progressPercent}%)...
          {:else if completedVideo}
            Đã hoàn thành video
          {:else}
            Hệ thống sẵn sàng
          {/if}
        </span>
      </div>
    </div>

    <div class="nav-right">
      <div class="engine-badge">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="12" cy="12" r="10"></circle>
          <polyline points="12 6 12 12 16 14"></polyline>
        </svg>
        <span>9router Gemini 3.8</span>
      </div>
    </div>
  </header>

  <!-- Main Studio Workspace -->
  <main class="studio-workspace">
    <!-- LEFT PANEL -->
    <section class="left-panel">
      <!-- Section 1: Ingest Article URL -->
      <div class="panel-card">
        <div class="card-header">
          <div class="step-num">1</div>
          <div>
            <h3>Nguồn Bài Báo</h3>
            <p>Dán link bài báo từ VnExpress, Tuổi Trẻ, Dân Trí, Thanh Niên...</p>
          </div>
        </div>

        <div class="url-input-wrap">
          <div class="input-icon">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
              <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
            </svg>
          </div>
          <input 
            type="url" 
            placeholder="Dán link bài báo vào đây..." 
            bind:value={articleUrl} 
            disabled={isCrawling || isRendering} 
          />
        </div>

        <div class="ingest-actions">
          <button 
            class="btn-secondary" 
            on:click={handlePreview} 
            disabled={isCrawling || isRendering}
          >
            {#if isCrawling}
              <span class="btn-spinner"></span>
              <span>Đang bóc tách...</span>
            {:else}
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <span>Bóc Tách & Xem Trước</span>
            {/if}
          </button>

          <button 
            class="btn-quick" 
            on:click={handleQuickRun} 
            disabled={isCrawling || isRendering}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"></polygon>
            </svg>
            <span>Tự Động 1-Click</span>
          </button>
        </div>
      </div>

      <!-- Section 2: Headline & Script Inspector -->
      <div class="panel-card">
        <div class="card-header">
          <div class="step-num">2</div>
          <div>
            <h3>Tiêu Đề & Kịch Bản Giật Tít</h3>
            <p>2 dòng tiêu đề in hoa đập vào mắt người xem trong 2 giây đầu</p>
          </div>
        </div>

        <div class="headline-grid">
          <div class="form-field">
            <label for="h1">Dòng 1 (Trắng, Viết Hoa):</label>
            <input 
              id="h1" 
              type="text" 
              placeholder="VD: TIN VUI CỰC MẠNH CHO HÀNG TRIỆU SINH VIÊN" 
              bind:value={headline1} 
              disabled={isRendering} 
            />
          </div>

          <div class="form-field">
            <label for="h2">Dòng 2 (Vàng Chanh, Từ Khóa Chấn Động):</label>
            <input 
              id="h2" 
              type="text" 
              placeholder="VD: CHÍNH THỨC CẤM ĐẠI HỌC ÉP NỘP TIỀN CẢ KỲ" 
              bind:value={headline2} 
              disabled={isRendering} 
            />
          </div>
        </div>

        <div class="form-field">
          <div class="field-title-row">
            <label for="script">Kịch Bản Lời Bình Thời Sự:</label>
            <span class="char-count">{fullScript.length} ký tự</span>
          </div>
          <textarea 
            id="script" 
            rows="3" 
            placeholder="Lời bình ngắn gọn thời sự 25-35 giây..." 
            bind:value={fullScript} 
            disabled={isRendering}
          ></textarea>
        </div>

        <!-- Filmstrip: Extracted Images -->
        <div class="form-field">
          <span class="field-label-text">Hình Ảnh Minh Họa Bóc Tách Được ({images.length} ảnh):</span>
          {#if images.length > 0}
            <div class="filmstrip-grid">
              {#each images as img}
                <div class="filmstrip-thumb active" title={img}>
                  <img src="file://{img.replace(/\\/g, '/')}" alt="thumb" />
                  <button class="thumb-remove" on:click={() => toggleImage(img)} title="Bỏ chọn ảnh này">×</button>
                </div>
              {/each}
            </div>
          {:else}
            <div class="empty-filmstrip">
              <span>Chưa có hình ảnh. Bấm "Bóc Tách & Xem Trước" để tải ảnh bài báo.</span>
            </div>
          {/if}
        </div>
      </div>

      <!-- Section 3: Color Picker & Styling -->
      <div class="panel-card">
        <div class="card-header">
          <div class="step-num">3</div>
          <div>
            <h3>Tùy Chọn Màu Nền & Giọng Đọc</h3>
            <p>Tự do chọn bất kỳ màu nào cho khung tiêu đề dưới đáy</p>
          </div>
        </div>

        <!-- Custom Color Picker Section -->
        <div class="form-field">
          <span class="field-label-text">Màu Nền Khung Tiêu Đề Dưới:</span>
          <div class="color-picker-row">
            <!-- Native Color Input -->
            <div class="color-input-box" style="background: {footerColor}">
              <input type="color" bind:value={footerColor} disabled={isRendering} />
            </div>

            <!-- Hex Textbox -->
            <input 
              class="hex-input" 
              type="text" 
              bind:value={footerColor} 
              placeholder="#EA580C" 
              disabled={isRendering} 
            />

            <!-- Quick Presets -->
            <div class="preset-chips">
              {#each colorPresets as p}
                <button 
                  class="preset-btn {footerColor.toUpperCase() === p.hex.toUpperCase() ? 'active' : ''}" 
                  style="--color: {p.hex}" 
                  on:click={() => footerColor = p.hex}
                  disabled={isRendering}
                  title={p.label}
                ></button>
              {/each}
            </div>
          </div>
        </div>

        <div class="options-grid">
          <div class="option-col">
            <span class="sub-label">Giọng Đọc Thời Sự:</span>
            <div class="chips-stack">
              {#each voices as v}
                <button 
                  class="chip-button {voiceName === v.id ? 'active' : ''}" 
                  on:click={() => voiceName = v.id}
                  disabled={isRendering}
                >
                  <div class="chip-text">
                    <strong>{v.name}</strong>
                    <small>{v.desc}</small>
                  </div>
                  <span class="chip-tag">{v.tag}</span>
                </button>
              {/each}
            </div>
          </div>

          <div class="option-col">
            <span class="sub-label">Khung Hình:</span>
            <div class="ratio-chips">
              <button 
                class="ratio-chip {ratio === '9:16' ? 'active' : ''}" 
                on:click={() => ratio = '9:16'}
                disabled={isRendering}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="5" y="2" width="14" height="20" rx="2" ry="2"></rect>
                  <line x1="12" y1="18" x2="12.01" y2="18"></line>
                </svg>
                <span>9:16 (Shorts/TikTok)</span>
              </button>

              <button 
                class="ratio-chip {ratio === '16:9' ? 'active' : ''}" 
                on:click={() => ratio = '16:9'}
                disabled={isRendering}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
                  <line x1="8" y1="21" x2="16" y2="21"></line>
                  <line x1="12" y1="17" x2="12" y2="21"></line>
                </svg>
                <span>16:9 (YouTube)</span>
              </button>
            </div>
          </div>
        </div>

        <!-- Big Launch Button -->
        <button 
          class="btn-primary-launch {isRendering ? 'rendering' : ''}" 
          on:click={handleRender} 
          disabled={isRendering || isCrawling}
        >
          {#if isRendering}
            <span class="btn-spinner large"></span>
            <span>Đang Dựng Video ({progressPercent}%)...</span>
          {:else}
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polygon points="5 3 19 12 5 21 5 3"></polygon>
            </svg>
            <span>DỰNG VIDEO TIN TỨC NGAY</span>
          {/if}
        </button>

        {#if errorMessage}
          <div class="error-box">
            <span>{errorMessage}</span>
          </div>
        {/if}
      </div>
    </section>

    <!-- RIGHT PANEL: PHONE CANVAS PREVIEW & STEPPER -->
    <section class="right-panel">
      <!-- 4-Node Visual Pipeline Graph -->
      <div class="stepper-card">
        <div class="stepper-nodes">
          <div class="step-node {progressStep === 'crawling' ? 'active' : (progressPercent > 20 ? 'done' : '')}">
            <div class="node-circle">
              {#if progressPercent > 20}✓{:else}1{/if}
            </div>
            <span>Bóc Tách</span>
          </div>
          <div class="step-line {progressPercent >= 40 ? 'filled' : ''}"></div>

          <div class="step-node {progressStep === 'summarizing' ? 'active' : (progressPercent >= 60 ? 'done' : '')}">
            <div class="node-circle">
              {#if progressPercent >= 60}✓{:else}2{/if}
            </div>
            <span>Biên Tập AI</span>
          </div>
          <div class="step-line {progressPercent >= 70 ? 'filled' : ''}"></div>

          <div class="step-node {progressStep === 'tts' ? 'active' : (progressPercent >= 75 ? 'done' : '')}">
            <div class="node-circle">
              {#if progressPercent >= 75}✓{:else}3{/if}
            </div>
            <span>Lồng Tiếng</span>
          </div>
          <div class="step-line {progressPercent >= 100 ? 'filled' : ''}"></div>

          <div class="step-node {progressStep === 'rendering' ? 'active' : (progressStep === 'complete' ? 'done' : '')}">
            <div class="node-circle">
              {#if progressStep === 'complete'}✓{:else}4{/if}
            </div>
            <span>FFmpeg HD</span>
          </div>
        </div>

        <div class="stepper-ticker">
          <div class="progress-bar-track">
            <div class="progress-bar-val" style="width: {progressPercent}%"></div>
          </div>
          <div class="ticker-text">
            <span>{progressLog}</span>
            <strong>{progressPercent}%</strong>
          </div>
        </div>
      </div>

      <!-- Smartphone Preview Frame Mockup -->
      <div class="mockup-container">
        <div class="phone-frame">
          <div class="phone-notch">
            <div class="notch-camera"></div>
          </div>

          <div class="phone-screen">
            {#if completedVideo}
              <div class="video-result-screen">
                <div class="result-badge">
                  <span>✓ Xuất Video Xong!</span>
                </div>
                <h4>Thành Phẩm 1080x1920 HD</h4>
                <p class="result-path">{completedVideo}</p>

                <div class="result-actions">
                  <button class="btn-primary" on:click={openOutputFolder}>
                    <span>Mở Thư Mục Video</span>
                  </button>
                  <button class="btn-subtle" on:click={() => navigator.clipboard.writeText(completedVideo)}>
                    Sao Chép Đường Dẫn
                  </button>
                </div>
              </div>
            {:else}
              <!-- Realistic 60/40 Theanh28 Split Layout Simulator -->
              <div class="simulator-layout">
                <!-- TOP 60%: Footage Viewer -->
                <div class="sim-top-footage">
                  {#if images.length > 0}
                    <img src="file://{images[0].replace(/\\/g, '/')}" alt="footage" />
                  {:else}
                    <div class="sim-placeholder-text">
                      <span>Vùng Video / Ảnh Báo Chí (60%)</span>
                    </div>
                  {/if}
                </div>

                <!-- BOTTOM 40%: Theanh28 Lower-third Banner -->
                <div class="sim-bottom-footer" style="background: {footerColor}">
                  <div class="sim-footer-tag">● TIN NÓNG HÔM NAY</div>
                  
                  <div class="sim-headline-1">
                    {headline1 ? headline1 : 'TIN VUI CỰC MẠNH CHO HÀNG TRIỆU SINH VIÊN'}
                  </div>

                  <div class="sim-headline-2">
                    {headline2 ? headline2 : 'CHÍNH THỨC CẤM ĐẠI HỌC ÉP ĐÓNG HỌC PHÍ GỘP'}
                  </div>

                  <div class="sim-lead-sub">
                    {fullScript ? fullScript.slice(0, 50) + '...' : 'Bộ Giáo dục yêu cầu thu học phí theo tháng, tối đa 10 tháng/năm...'}
                  </div>
                </div>
              </div>
            {/if}
          </div>
        </div>
      </div>
    </section>
  </main>
</div>

<style>
  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background: #f8fafc;
    color: #1e293b;
    -webkit-font-smoothing: antialiased;
  }

  .studio-app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background: #f8fafc;
  }

  /* TOP NAVIGATION */
  .top-nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 28px;
    background: #ffffff;
    border-bottom: 1px solid #e2e8f0;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
    position: sticky;
    top: 0;
    z-index: 50;
  }

  .nav-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .logo-mark {
    width: 38px;
    height: 38px;
    background: #2563eb;
    color: #ffffff;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 4px 10px rgba(37, 99, 235, 0.25);
  }

  .brand-text {
    display: flex;
    flex-direction: column;
  }

  .app-title {
    font-size: 17px;
    font-weight: 700;
    color: #0f172a;
    letter-spacing: -0.3px;
  }

  .app-badge {
    font-size: 11px;
    font-weight: 600;
    color: #2563eb;
    letter-spacing: 0.2px;
  }

  .status-pill {
    display: flex;
    align-items: center;
    gap: 8px;
    background: #f1f5f9;
    padding: 6px 14px;
    border-radius: 9999px;
    font-size: 13px;
    color: #475569;
    border: 1px solid #e2e8f0;
  }

  .pulse-indicator {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #10b981;
  }

  .pulse-indicator.active {
    background: #f59e0b;
    box-shadow: 0 0 0 3px rgba(245, 158, 11, 0.2);
    animation: pulseAnim 1s infinite alternate;
  }

  @keyframes pulseAnim {
    from { opacity: 0.4; }
    to { opacity: 1; }
  }

  .engine-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    color: #64748b;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    padding: 6px 12px;
    border-radius: 8px;
    font-weight: 500;
  }

  /* WORKSPACE GRID */
  .studio-workspace {
    display: grid;
    grid-template-columns: 1.15fr 0.85fr;
    gap: 24px;
    padding: 24px 28px;
    max-width: 1350px;
    margin: 0 auto;
    width: 100%;
    box-sizing: border-box;
  }

  /* PANEL CARDS */
  .panel-card {
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    padding: 20px;
    margin-bottom: 20px;
    box-shadow: 0 4px 16px -2px rgba(0, 0, 0, 0.03);
  }

  .card-header {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    margin-bottom: 16px;
  }

  .step-num {
    width: 26px;
    height: 26px;
    background: #eff6ff;
    color: #2563eb;
    border: 1px solid #bfdbfe;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .card-header h3 {
    margin: 0 0 2px;
    font-size: 16px;
    font-weight: 700;
    color: #0f172a;
  }

  .card-header p {
    margin: 0;
    font-size: 12px;
    color: #64748b;
  }

  /* URL INPUT */
  .url-input-wrap {
    position: relative;
    display: flex;
    align-items: center;
    margin-bottom: 14px;
  }

  .input-icon {
    position: absolute;
    left: 12px;
    color: #94a3b8;
    display: flex;
    align-items: center;
  }

  .url-input-wrap input {
    width: 100%;
    padding: 11px 14px 11px 38px;
    border: 1px solid #cbd5e1;
    border-radius: 10px;
    font-size: 14px;
    color: #0f172a;
    background: #ffffff;
    outline: none;
    transition: all 0.2s;
  }

  .url-input-wrap input:focus {
    border-color: #2563eb;
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
  }

  .ingest-actions {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .btn-secondary {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    background: #f1f5f9;
    color: #1e293b;
    border: 1px solid #cbd5e1;
    border-radius: 10px;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-secondary:hover:not(:disabled) {
    background: #e2e8f0;
    border-color: #94a3b8;
  }

  .btn-quick {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    background: #eff6ff;
    color: #1d4ed8;
    border: 1px solid #bfdbfe;
    border-radius: 10px;
    padding: 10px 16px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-quick:hover:not(:disabled) {
    background: #dbeafe;
  }

  /* HEADLINES GRID */
  .headline-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
    margin-bottom: 14px;
  }

  .form-field {
    margin-bottom: 14px;
  }

  .field-label-text, .form-field label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    color: #475569;
    margin-bottom: 6px;
  }

  .field-title-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .char-count {
    font-size: 11px;
    color: #94a3b8;
  }

  .form-field input, .form-field textarea {
    width: 100%;
    padding: 10px 14px;
    border: 1px solid #cbd5e1;
    border-radius: 10px;
    font-size: 13px;
    color: #0f172a;
    background: #ffffff;
    box-sizing: border-box;
    outline: none;
    transition: all 0.2s;
    font-family: inherit;
  }

  .form-field input:focus, .form-field textarea:focus {
    border-color: #2563eb;
    box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
  }

  /* COLOR PICKER ROW */
  .color-picker-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .color-input-box {
    width: 40px;
    height: 40px;
    border-radius: 8px;
    border: 2px solid #cbd5e1;
    overflow: hidden;
    position: relative;
    cursor: pointer;
    box-shadow: 0 2px 5px rgba(0,0,0,0.1);
  }

  .color-input-box input[type="color"] {
    position: absolute;
    top: -10px;
    left: -10px;
    width: 60px;
    height: 60px;
    opacity: 0;
    cursor: pointer;
  }

  .hex-input {
    width: 110px !important;
    text-transform: uppercase;
    font-weight: 700;
    font-family: monospace;
  }

  .preset-chips {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .preset-btn {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: 2px solid #ffffff;
    background: var(--color);
    box-shadow: 0 0 0 1px #cbd5e1;
    cursor: pointer;
    transition: transform 0.15s;
  }

  .preset-btn:hover {
    transform: scale(1.15);
  }

  .preset-btn.active {
    box-shadow: 0 0 0 2px #2563eb;
    transform: scale(1.1);
  }

  /* FILMSTRIP */
  .filmstrip-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 10px;
    margin-top: 6px;
  }

  .filmstrip-thumb {
    position: relative;
    aspect-ratio: 9/16;
    border-radius: 8px;
    overflow: hidden;
    border: 2px solid #2563eb;
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
  }

  .filmstrip-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .thumb-remove {
    position: absolute;
    top: 4px;
    right: 4px;
    width: 20px;
    height: 20px;
    background: rgba(0, 0, 0, 0.6);
    color: #ffffff;
    border: none;
    border-radius: 50%;
    cursor: pointer;
    font-size: 14px;
    line-height: 1;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .empty-filmstrip {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    border: 2px dashed #cbd5e1;
    border-radius: 10px;
    color: #64748b;
    font-size: 12px;
  }

  /* OPTIONS GRID */
  .options-grid {
    display: grid;
    grid-template-columns: 1.2fr 1fr;
    gap: 16px;
    margin-bottom: 20px;
    margin-top: 14px;
  }

  .sub-label {
    display: block;
    font-size: 12px;
    font-weight: 600;
    color: #475569;
    margin-bottom: 8px;
  }

  .chips-stack {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .chip-button {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    background: #ffffff;
    cursor: pointer;
    text-align: left;
    transition: all 0.2s;
  }

  .chip-button.active {
    border-color: #2563eb;
    background: #eff6ff;
  }

  .chip-text strong {
    display: block;
    font-size: 12px;
    color: #0f172a;
  }

  .chip-text small {
    font-size: 11px;
    color: #64748b;
  }

  .chip-tag {
    font-size: 10px;
    font-weight: 600;
    background: #e2e8f0;
    color: #475569;
    padding: 2px 6px;
    border-radius: 4px;
  }

  .ratio-chips {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .ratio-chip {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    background: #ffffff;
    font-size: 12px;
    font-weight: 600;
    color: #1e293b;
    cursor: pointer;
    transition: all 0.2s;
  }

  .ratio-chip.active {
    border-color: #2563eb;
    background: #eff6ff;
    color: #1d4ed8;
  }

  /* LAUNCH BUTTON */
  .btn-primary-launch {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 10px;
    background: #2563eb;
    color: #ffffff;
    border: none;
    border-radius: 12px;
    padding: 15px;
    font-size: 15px;
    font-weight: 700;
    cursor: pointer;
    box-shadow: 0 4px 14px rgba(37, 99, 235, 0.3);
    transition: all 0.2s;
  }

  .btn-primary-launch:hover:not(:disabled) {
    background: #1d4ed8;
    transform: translateY(-1px);
  }

  .btn-primary-launch:disabled {
    opacity: 0.65;
    cursor: not-allowed;
  }

  .error-box {
    margin-top: 14px;
    padding: 10px 14px;
    background: #fef2f2;
    border: 1px solid #fecaca;
    color: #b91c1c;
    border-radius: 8px;
    font-size: 13px;
  }

  /* RIGHT PANEL */
  .right-panel {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .stepper-card {
    background: #ffffff;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    padding: 18px 20px;
    box-shadow: 0 4px 16px -2px rgba(0, 0, 0, 0.03);
  }

  .stepper-nodes {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 14px;
  }

  .step-node {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 4px;
  }

  .node-circle {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: #f1f5f9;
    border: 2px solid #cbd5e1;
    color: #64748b;
    font-size: 12px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .step-node span {
    font-size: 11px;
    font-weight: 600;
    color: #64748b;
  }

  .step-node.active .node-circle {
    background: #eff6ff;
    border-color: #2563eb;
    color: #2563eb;
    box-shadow: 0 0 0 4px rgba(37, 99, 235, 0.15);
  }

  .step-node.active span {
    color: #2563eb;
  }

  .step-node.done .node-circle {
    background: #10b981;
    border-color: #10b981;
    color: #ffffff;
  }

  .step-node.done span {
    color: #0f172a;
  }

  .step-line {
    flex: 1;
    height: 2px;
    background: #e2e8f0;
    margin: 0 8px 16px;
  }

  .step-line.filled {
    background: #10b981;
  }

  .progress-bar-track {
    height: 6px;
    background: #f1f5f9;
    border-radius: 9999px;
    overflow: hidden;
    margin-bottom: 6px;
  }

  .progress-bar-val {
    height: 100%;
    background: #2563eb;
    border-radius: 9999px;
    transition: width 0.3s ease;
  }

  .ticker-text {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #64748b;
  }

  .ticker-text strong {
    color: #0f172a;
  }

  /* REALISTIC 60/40 PHONE MOCKUP */
  .mockup-container {
    display: flex;
    justify-content: center;
  }

  .phone-frame {
    width: 290px;
    height: 520px;
    background: #ffffff;
    border: 8px solid #cbd5e1;
    border-radius: 36px;
    box-shadow: 0 16px 36px -8px rgba(0, 0, 0, 0.12);
    position: relative;
    overflow: hidden;
    display: flex;
    flex-direction: column;
  }

  .phone-notch {
    height: 18px;
    display: flex;
    justify-content: center;
    align-items: center;
    background: #cbd5e1;
  }

  .notch-camera {
    width: 48px;
    height: 4px;
    background: #94a3b8;
    border-radius: 9999px;
  }

  .phone-screen {
    flex: 1;
    background: #0f172a;
    position: relative;
    overflow: hidden;
    display: flex;
  }

  .simulator-layout {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  /* TOP 60% */
  .sim-top-footage {
    height: 60%;
    width: 100%;
    background: #020617;
    position: relative;
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .sim-top-footage img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .sim-placeholder-text {
    color: #64748b;
    font-size: 11px;
    text-align: center;
    padding: 20px;
  }

  /* BOTTOM 40% (THEANH28 STYLE) */
  .sim-bottom-footer {
    height: 40%;
    width: 100%;
    padding: 12px 14px;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    justify-content: flex-start;
    transition: background 0.25s ease;
  }

  .sim-footer-tag {
    background: rgba(0, 0, 0, 0.35);
    color: #ffffff;
    font-size: 9px;
    font-weight: 700;
    padding: 2px 6px;
    border-radius: 4px;
    display: inline-block;
    width: fit-content;
    margin-bottom: 6px;
    letter-spacing: 0.3px;
  }

  .sim-headline-1 {
    color: #ffffff;
    font-size: 11px;
    font-weight: 900;
    line-height: 1.25;
    margin-bottom: 4px;
    text-transform: uppercase;
  }

  .sim-headline-2 {
    color: #fef08a;
    font-size: 11px;
    font-weight: 900;
    line-height: 1.25;
    margin-bottom: 6px;
    text-transform: uppercase;
  }

  .sim-lead-sub {
    color: #f8fafc;
    font-size: 9.5px;
    line-height: 1.3;
    opacity: 0.95;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  /* VIDEO FINISHED */
  .video-result-screen {
    width: 100%;
    height: 100%;
    background: #ffffff;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px;
    box-sizing: border-box;
    text-align: center;
  }

  .result-badge {
    background: #ecfdf5;
    color: #059669;
    padding: 8px 16px;
    border-radius: 9999px;
    font-size: 13px;
    font-weight: 700;
    margin-bottom: 12px;
  }

  .video-result-screen h4 {
    margin: 0 0 8px;
    font-size: 15px;
    color: #0f172a;
  }

  .result-path {
    background: #f1f5f9;
    padding: 8px;
    border-radius: 6px;
    font-size: 10px;
    color: #475569;
    word-break: break-all;
    margin-bottom: 16px;
  }

  .result-actions {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
  }

  .result-actions .btn-primary {
    background: #2563eb;
    color: #ffffff;
    border: none;
    padding: 10px;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  .result-actions .btn-subtle {
    background: #f1f5f9;
    color: #475569;
    border: 1px solid #cbd5e1;
    padding: 8px;
    border-radius: 8px;
    font-size: 12px;
    cursor: pointer;
  }

  .btn-spinner {
    width: 14px;
    height: 14px;
    border: 2px solid rgba(0, 0, 0, 0.2);
    border-top-color: currentColor;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  .btn-spinner.large {
    width: 18px;
    height: 18px;
    border-width: 3px;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>