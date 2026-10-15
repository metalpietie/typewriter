import 'package:flutter/material.dart';

import 'app.dart';
import 'ui/notes_page.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final app = await AppServices.create();
  runApp(TypewriterNotes(app: app));
  app.syncNow(quiet: true); // a quiet try when the app opens: it does nothing if the computer is not there
}

class TypewriterNotes extends StatelessWidget {
  const TypewriterNotes({super.key, required this.app});
  final AppServices app;

  // the same indigo as Typewriter
  static const accent = Color(0xFF8082FF);

  ThemeData _theme(Brightness b) {
    final dark = b == Brightness.dark;
    final scheme = ColorScheme.fromSeed(seedColor: accent, brightness: b).copyWith(
      surface: dark ? const Color(0xFF0F1014) : const Color(0xFFF5F5FA),
      primary: dark ? accent : const Color(0xFF5B5BD6),
    );
    return ThemeData(
      colorScheme: scheme,
      useMaterial3: true,
      scaffoldBackgroundColor: scheme.surface,
      appBarTheme: AppBarTheme(backgroundColor: scheme.surface, surfaceTintColor: Colors.transparent),
      cardTheme: CardThemeData(color: dark ? const Color(0xFF16171D) : Colors.white, elevation: 0, shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12), side: BorderSide(color: dark ? const Color(0xFF262833) : const Color(0xFFE4E4EF)))),
      inputDecorationTheme: InputDecorationTheme(border: OutlineInputBorder(borderRadius: BorderRadius.circular(10))),
    );
  }

  @override
  Widget build(BuildContext context) => MaterialApp(
        title: 'Typewriter Notes',
        debugShowCheckedModeBanner: false,
        theme: _theme(Brightness.light),
        darkTheme: _theme(Brightness.dark),
        home: NotesPage(app: app),
      );
}
