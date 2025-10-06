<template>
  <header
    style="
      display: flex;
      gap: 12px;
      align-items: center;
      padding: 12px 16px;
      border-bottom: 1px solid #eee;
    "
  >
    <h1 style="margin: 0; font-size: 18px">TrackAndFeel</h1>
    <nav style="display: flex; gap: 10px">
      <router-link to="/activities">Activities</router-link>
      <router-link to="/upload">Upload</router-link>
    </nav>
  </header>
  <main style="padding: 16px">
    <router-view />
  </main>
</template>
